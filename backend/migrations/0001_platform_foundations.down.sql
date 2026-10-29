-- Development/test only: production migrations are forward-only (doc 06 §10, expand → migrate → contract).
DROP SCHEMA platform CASCADE;
