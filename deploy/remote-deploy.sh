#!/usr/bin/env bash
# Runs ON the server, piped over ssh by .github/workflows/deploy-api.yml (same commit):
#   ssh host bash -s -- deploy|rollback <sha> < deploy/remote-deploy.sh
# The source arrives by rsync from the runner; there is no git and no GitHub key on the server.
# Order: lock → tag the running image as :previous → build :<sha> → dump the DB → swap → health → keep 5 tags.
# Any failure after the build puts :previous back.
set -Eeuo pipefail
mode=$1 sha=$2
app=familycotton-api                             # existing compose project: the volume is familycotton-api_pgdata
dir=/opt/familycotton/familycotton-api
img=familycotton-api-api
unset COMPOSE_PROJECT_NAME                       # an exported name would redirect every command to another project
compose() { docker compose -p "$app" -f "$dir/docker-compose.prod.yml" --env-file "$dir/.env.prod" "$@"; }

# `up -d` exits 0 for a crash-looping container: wait for the container healthcheck and our own probe.
healthy() {
  for _ in $(seq 60); do
    if [ "$(docker inspect -f '{{.State.Health.Status}}' familycotton-api 2>/dev/null)" = healthy ] &&
       curl -fsS --max-time 5 http://127.0.0.1:8082/api/v1/health | grep -q ok; then
      return 0
    fi
    sleep 2
  done
  return 1
}
swap() { API_TAG=$1 compose up -d --no-build api && healthy; }

# The script itself arrives on stdin (`bash -s`): anything that reads stdin (`compose exec -T`, `compose run`)
# would swallow the rest of it and the deploy would "succeed" without swapping. Everything runs inside
# main() with stdin closed; bash has parsed the whole function before the call line.
main() {
  exec 9>"/var/lock/$app-deploy.lock"
  flock -n 9 || { echo "another deploy of $app is running"; exit 1; }
  [ -s "$dir/.env.prod" ] || { echo "missing $dir/.env.prod"; exit 1; }

  if [ "$mode" = rollback ]; then
    docker image inspect "$img:previous" >/dev/null   # nothing to roll back to: fail loudly
    swap previous
    echo "rolled back to $img:previous"
    exit 0
  fi

  compose config -q
  # Docker 29 + containerd image store deletes an untagged image at once, even under a running container:
  # tag the running image BEFORE building, or there is nothing to roll back to.
  running=$(docker inspect -f '{{.Image}}' familycotton-api 2>/dev/null || true)
  if [ -n "$running" ]; then docker tag "$running" "$img:previous"; fi

  API_TAG=$sha compose build api

  # Dump before the new version can migrate anything.
  if [ -n "$(compose ps -q db 2>/dev/null)" ]; then
    install -d -m 700 /var/backups/familycotton/pre-deploy
    compose exec -T db sh -c 'pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB"' | gzip > "/var/backups/familycotton/pre-deploy/$sha.sql.gz"
    ls -1t /var/backups/familycotton/pre-deploy/*.sql.gz | tail -n +6 | xargs -r rm -f
  fi

  if ! swap "$sha"; then
    echo "::error::new version unhealthy"   # no container logs here: this repo is public and so are its Actions logs
    if docker image inspect "$img:previous" >/dev/null 2>&1 && swap previous; then
      echo "ROLLED BACK to $img:previous"
    fi
    exit 1
  fi
  docker tag "$img:$sha" "$img:latest"            # a manual `compose up` without API_TAG gets what is deployed
  # Retention: the 5 newest tags; latest and previous always stay.
  docker images "$img" --format '{{.CreatedAt}}\t{{.Tag}}' | sort -r | awk -F'\t' 'NR>5{print $2}' \
    | grep -vE '^(latest|previous)$' | xargs -r -I{} docker rmi "$img:{}" >/dev/null 2>&1 || true
  echo "deployed $sha"
}

main "$@" </dev/null
