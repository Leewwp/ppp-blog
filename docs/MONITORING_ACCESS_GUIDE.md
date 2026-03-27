# Monitoring Access Guide

## Target State

- Public ports: `22`, `80`, `443`
- Optional public port: `18090` only if you still use the validation instance in CI/CD
- Internal-only ports: `3000`, `9090`, `3100`, `8090`, `8091`, `8092`, `8093`, `8094`, `6379`, `9092`, `3307`

This repository now binds Grafana, Prometheus, Loki, Halo, and the internal Go services to `127.0.0.1` in `docker-compose.yml`. That means:

- The public website still goes through Nginx on `80/443`
- Grafana should be exposed through an Nginx subdomain such as `grafana.leeppp.online`
- Prometheus should stay private and be accessed through an SSH tunnel

## Files Added For This Workflow

- `monitoring/nginx/grafana-http.conf.example`
- `monitoring/nginx/grafana-https.conf.example`

Use the HTTP example before the TLS certificate exists. Use the HTTPS example after `certbot` has issued the certificate.

## Step 1: Update The Server Code

On the server:

```bash
cd /opt/halo-blog
git pull
docker compose up -d
```

If you do not deploy from Git on the server, upload the updated files manually:

- `docker-compose.yml`
- `docs/MONITORING_ACCESS_GUIDE.md`
- `monitoring/nginx/grafana-http.conf.example`
- `monitoring/nginx/grafana-https.conf.example`

## Step 2: Confirm The Required Environment Variables

Open `/opt/halo-blog/.env` and verify the following values.

```bash
cd /opt/halo-blog
nano .env
```

Required manual values:

| Variable | What to set | Example |
|---|---|---|
| `HALO_EXTERNAL_URL` | Your public blog URL | `https://leeppp.online` |
| `GF_SECURITY_ADMIN_PASSWORD` | Grafana admin password, must not stay as `change_me` | `replace_with_a_long_random_password` |
| `SERVER_HOST` | Your server public IP, used by deployment workflows | `43.154.x.x` |
| `SSH_PORT` | Your SSH port | `22` |

Notes:

- `GF_SECURITY_ADMIN_PASSWORD` is the Grafana login password, not the Nginx Basic Auth password.
- Use a different password for Grafana and Nginx Basic Auth.

After saving `.env`, restart the containers:

```bash
cd /opt/halo-blog
docker compose up -d
```

## Step 3: Verify Internal Services From The Server Itself

Run these commands on the server:

```bash
curl http://127.0.0.1:3000/login
curl http://127.0.0.1:9090/-/ready
curl http://127.0.0.1:8090/actuator/health/readiness
docker compose ps
```

Expected results:

- `127.0.0.1:3000` returns the Grafana login page HTML
- `127.0.0.1:9090/-/ready` returns `Prometheus is Ready.`
- `127.0.0.1:8090/actuator/health/readiness` returns a JSON payload containing `"status":"UP"`

If any of these fail, fix the container health issue before touching Nginx.

## Step 4: Add The Grafana DNS Record

Create a new DNS `A` record:

| Host | Type | Value |
|---|---|---|
| `grafana` | `A` | Your server public IP |

Result:

- `grafana.leeppp.online` resolves to the same server as `leeppp.online`

Wait until DNS resolution is effective, then verify:

```bash
ping grafana.leeppp.online
```

## Step 5: Install The Nginx Helper Package

This is needed for `htpasswd`.

```bash
sudo apt update
sudo apt install -y apache2-utils
```

## Step 6: Create The Temporary HTTP Nginx Site

Copy the HTTP template to the Nginx site directory:

```bash
sudo cp /opt/halo-blog/monitoring/nginx/grafana-http.conf.example /etc/nginx/sites-available/grafana
sudo nano /etc/nginx/sites-available/grafana
```

Replace these placeholders manually:

| Placeholder | What to replace it with | Example |
|---|---|---|
| `__GRAFANA_SERVER_NAME__` | Your Grafana subdomain | `grafana.leeppp.online` |
| `__GRAFANA_UPSTREAM__` | Grafana local upstream | `http://127.0.0.1:3000` |

Enable the site:

```bash
sudo ln -s /etc/nginx/sites-available/grafana /etc/nginx/sites-enabled/grafana
sudo nginx -t
sudo systemctl reload nginx
```

If the symlink already exists, skip the `ln -s` command.

## Step 7: Issue The TLS Certificate

Run `certbot` after the HTTP site is active:

```bash
sudo certbot --nginx -d grafana.leeppp.online
```

Manual values in this command:

| Part | What to replace it with |
|---|---|
| `grafana.leeppp.online` | Your actual Grafana subdomain |

After success, the certificate files will usually be:

- `/etc/letsencrypt/live/grafana.leeppp.online/fullchain.pem`
- `/etc/letsencrypt/live/grafana.leeppp.online/privkey.pem`

## Step 8: Create The Nginx Basic Auth File

Create the password file that protects Grafana before the Grafana login page is reached:

```bash
sudo htpasswd -c /etc/nginx/.htpasswd_grafana __GRAFANA_BASIC_AUTH_USER__
```

Replace this placeholder manually:

| Placeholder | What to replace it with | Example |
|---|---|---|
| `__GRAFANA_BASIC_AUTH_USER__` | Your Nginx Basic Auth username | `admin_monitor` |

You will then be prompted to enter the Nginx Basic Auth password interactively.

Recommended rule:

- Nginx Basic Auth username: a human-readable admin account such as `admin_monitor`
- Nginx Basic Auth password: a strong password different from `GF_SECURITY_ADMIN_PASSWORD`

## Step 9: Switch Nginx To The Final HTTPS Config

Replace the temporary HTTP site with the final HTTPS template:

```bash
sudo cp /opt/halo-blog/monitoring/nginx/grafana-https.conf.example /etc/nginx/sites-available/grafana
sudo nano /etc/nginx/sites-available/grafana
```

Replace these placeholders manually:

| Placeholder | What to replace it with | Example |
|---|---|---|
| `__GRAFANA_SERVER_NAME__` | Your Grafana subdomain | `grafana.leeppp.online` |
| `__SSL_CERT_PATH__` | TLS certificate full chain path | `/etc/letsencrypt/live/grafana.leeppp.online/fullchain.pem` |
| `__SSL_CERT_KEY_PATH__` | TLS private key path | `/etc/letsencrypt/live/grafana.leeppp.online/privkey.pem` |
| `__GRAFANA_BASIC_AUTH_FILE__` | Nginx Basic Auth file path | `/etc/nginx/.htpasswd_grafana` |
| `__GRAFANA_UPSTREAM__` | Grafana local upstream | `http://127.0.0.1:3000` |

Validate and reload Nginx:

```bash
sudo nginx -t
sudo systemctl reload nginx
```

## Step 10: Final Access Verification

From your own computer:

```bash
curl -I https://grafana.leeppp.online
```

Then open:

- `https://grafana.leeppp.online`

Expected access flow:

1. Nginx Basic Auth prompt appears first
2. After Basic Auth succeeds, Grafana login page appears
3. You log in with:
   - Grafana username: `admin`
   - Grafana password: the value of `GF_SECURITY_ADMIN_PASSWORD` in `/opt/halo-blog/.env`

## Step 11: Access Prometheus Through SSH Tunnel

Do not reopen port `9090` to the public internet. Access it through an SSH tunnel from your local computer:

```bash
ssh -L 9090:127.0.0.1:9090 root@__SERVER_IP__
```

If your SSH service uses a custom port:

```bash
ssh -p __SSH_PORT__ -L 9090:127.0.0.1:9090 root@__SERVER_IP__
```

Replace these placeholders manually:

| Placeholder | What to replace it with | Example |
|---|---|---|
| `__SERVER_IP__` | Your server public IP | `43.154.x.x` |
| `__SSH_PORT__` | Your SSH port | `22` |

After the tunnel is established, open this URL on your local computer:

- `http://127.0.0.1:9090`

## Optional: Emergency Grafana SSH Tunnel

If Nginx is broken but Grafana is still healthy on the server, you can temporarily bypass Nginx:

```bash
ssh -L 3000:127.0.0.1:3000 root@__SERVER_IP__
```

Then open:

- `http://127.0.0.1:3000`

## Final Firewall And Security Group Rules

Recommended public rules:

- Keep open: `22/tcp`, `80/tcp`, `443/tcp`
- Keep open only if validation CI/CD still needs it: `18090/tcp`
- Keep closed to the internet: `3000/tcp`, `9090/tcp`, `3100/tcp`, `8090/tcp`, `8091/tcp`, `8092/tcp`, `8093/tcp`, `8094/tcp`, `8081/tcp`

## Troubleshooting Commands

Run these on the server:

```bash
cd /opt/halo-blog
docker compose ps
docker compose logs --tail=100 grafana prometheus loki promtail
curl http://127.0.0.1:3000/login
curl http://127.0.0.1:9090/-/ready
sudo nginx -t
sudo systemctl status nginx
sudo tail -n 100 /var/log/nginx/grafana-error.log
```

Common failure points:

- `certbot` fails: DNS for `grafana.leeppp.online` is not pointing to the server yet
- `502 Bad Gateway`: Grafana container is down, or `proxy_pass` is not `http://127.0.0.1:3000`
- Grafana login fails after Basic Auth succeeds: `GF_SECURITY_ADMIN_PASSWORD` in `.env` is wrong or Grafana was not restarted
- `127.0.0.1:9090` works on the server but not on your computer: the SSH tunnel command was not kept open
