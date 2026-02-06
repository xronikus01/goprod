## Manual tests (PowerShell)

### Start DB
```powershell
docker compose up -d
docker compose ps
```
Run service
```powershell
go run .
```

Register
```powershell
Invoke-RestMethod -Method Post -Uri "http://localhost:8080/register" -ContentType "application/json" -Body '{"email":"u1@mail.com","username":"u1","password":"secret12"}'
```

Login (get token)
```powershell
$resp = Invoke-RestMethod -Method Post -Uri "http://localhost:8080/login" -ContentType "application/json" -Body '{"email":"u1@mail.com","password":"secret12"}'
$resp.token
```
Profile (protected)
```powershell
Invoke-RestMethod -Method Get -Uri "http://localhost:8080/profile" -Headers @{ Authorization = "Bearer $($resp.token)" }
```
Profile without token (should be 401)
```powershell
try { Invoke-RestMethod -Method Get -Uri "http://localhost:8080/profile" } catch { $_.Exception.Response.StatusCode.value__ }
```
Check bcrypt hash in DB
```powershell
docker exec -it secure_service_db psql -U postgres -d secure_service -c "select email, username, password_hash from users;"
```