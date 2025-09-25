
## Prepare before development

1. Create file .env or run this command in root project

```bash
  touch .env 
```

2. Set variable like in this example or you can follow in .env.example

```bash
GO_ENV=development
API_NAME=Go-Application-Template
PORT=8000
DB_URL=postgresql://postgres:$USERNAME@HOST:PORT/DATABASE
```

3. Open your terminal and run command Air package
```bash
air
```

4. Your terminal will see like this 
```bash
2025/09/25 14:32:17 ------ Running in 'development' mode... ------
2025/09/25 14:32:17 Successfully connected to database pgx ✅!

 ┌───────────────────────────────────────────────────┐ 
 │              Go-Application-Template              │ 
 │                   Fiber v2.52.4                   │ 
 │               http://127.0.0.1:8000               │ 
 │       (bound on host 0.0.0.0 and port 8000)       │ 
 │                                                   │ 
 │ Handlers ............ 13  Processes ........... 1 │ 
 │ Prefork ....... Disabled  PID ............. 77055 │ 
 └───────────────────────────────────────────────────┘ 
```

5. You can follow folder structure:
```bash
- cmd/api/main.go
- api/v1/*
```

## Thank you distributors and document at
- air: https://github.com/air-verse/air
- golang: https://go.dev/
- golang fiber framework: https://github.com/gofiber/fiber || https://docs.gofiber.io/

## If you have Questions you can contact me
- Githubs: https://github.com/Binh-2060
- Website: https://binh.phouservice.com
- Email: binhxayxana@gmail.com



    