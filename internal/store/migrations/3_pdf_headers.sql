CREATE TABLE pdf_headers (
  id bigserial PRIMARY KEY,
  name text NOT NULL,
  value text NOT NULL,
  sensitive boolean NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX pdf_headers_name_ci_idx ON pdf_headers (lower(name));
