add_header Content-Security-Policy "default-src 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'; form-action 'self'; script-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com data:; img-src 'self' data: blob: https://*.r2.cloudflarestorage.com; connect-src 'self'; media-src 'self' blob:" always;
add_header X-Content-Type-Options "nosniff" always;
add_header Referrer-Policy "same-origin" always;
add_header Permissions-Policy "camera=(self), microphone=(), geolocation=()" always;
add_header X-Frame-Options "DENY" always;
add_header X-Robots-Tag $admin_robots always;
