-- +goose Up
-- Fix old sales that have total_amount but all paid_* fields are zero.
-- These were created before the split-payment system was implemented.
-- Assumes they were cash payments.
UPDATE sales
SET paid_cash = total_amount
WHERE paid_cash = 0
  AND paid_terminal = 0
  AND paid_online = 0
  AND paid_debt = 0
  AND total_amount > 0;

-- +goose Down
-- Cannot reliably reverse this migration.
