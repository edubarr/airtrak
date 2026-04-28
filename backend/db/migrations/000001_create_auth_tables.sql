-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION airtrak_set_updated_at() RETURNS trigger AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TABLE users (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  display_name text NOT NULL CHECK (display_name <> ''),
  locale text NOT NULL DEFAULT 'pt-BR' CHECK (locale IN ('pt-BR', 'en')),
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);

CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION airtrak_set_updated_at();

CREATE TABLE user_credentials (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type text NOT NULL CHECK (type IN ('email_password', 'whatsapp_jid')),
  identifier text NOT NULL CHECK (identifier <> ''),
  secret_hash text,
  verified_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz,
  CHECK (
    (type = 'email_password' AND secret_hash IS NOT NULL)
    OR (type = 'whatsapp_jid' AND secret_hash IS NULL)
  )
);

CREATE INDEX user_credentials_user_id_idx ON user_credentials(user_id);
CREATE UNIQUE INDEX user_credentials_type_identifier_active_idx
  ON user_credentials(type, identifier)
  WHERE deleted_at IS NULL;

CREATE TRIGGER user_credentials_set_updated_at
BEFORE UPDATE ON user_credentials
FOR EACH ROW
EXECUTE FUNCTION airtrak_set_updated_at();

CREATE TABLE user_sessions (
  id uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at timestamptz NOT NULL,
  revoked_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX user_sessions_user_id_idx ON user_sessions(user_id);
CREATE INDEX user_sessions_active_idx ON user_sessions(user_id, expires_at) WHERE revoked_at IS NULL;

CREATE TRIGGER user_sessions_set_updated_at
BEFORE UPDATE ON user_sessions
FOR EACH ROW
EXECUTE FUNCTION airtrak_set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS user_credentials;
DROP TABLE IF EXISTS users;
DROP FUNCTION IF EXISTS airtrak_set_updated_at();
