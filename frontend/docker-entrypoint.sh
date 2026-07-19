#!/bin/sh
set -e

# Inject runtime environment variables for Next.js frontend
mkdir -p /app/public
cat <<EOF > /app/public/env-config.js
window.__ENV__ = {
  NEXT_PUBLIC_BACKEND_URL: "${NEXT_PUBLIC_BACKEND_URL}"
};
EOF

exec "$@"
