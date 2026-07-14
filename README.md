# BimaNyaya (बीमान्याय)

BimaNyaya is a high-fidelity, AI-powered insurance grievance redressal platform. It empowers policyholders to analyze unfair claim rejections, verify eligibility, check against legal/regulatory guidelines (IRDAI), auto-generate professional representations, and navigate the disputes process seamlessly.

---

## 🏗 System Architecture

The monorepo operates on a modular, decoupled full-stack architecture designed for performance, high availability, and type safety:

```text
       ┌────────────────────────┐
       │   TanStack Start Web   │
       │       (Frontend)       │
       └───────────┬────────────┘
                   │
         [ Clerk Auth Header ]
                   │
                   ▼
       ┌────────────────────────┐
       │      Go Core API       │ (Port 8080)
       │    (Orchestration)     │
       └──────┬──────────┬──────┘
              │          │
    [ Convex Client ]  [ HTTP Proxy ]
              │          │
              ▼          ▼
       ┌──────────┐  ┌──────────┐
       │  Convex  │  │ Python   │ (Port 8000)
       │ Database │  │ AI-Worker│
       └──────────┘  └──────────┘
```

1. **Frontend:** Built with TanStack Start, integrated with Clerk Authentication (Email OTP/Magic Links) and the Convex client.
2. **Go Core API:** Acts as the primary backend gateway, performing JWKS key-validation for Clerk JWTs, enforcing CORS allowlists, capping request sizes (10MB), and proxying client calls.
3. **Convex Backend:** Serves as the real-time database, workflows state machine, and serverless logic engine.
4. **Python AI-Worker:** Handles intensive ML/RAG pipelines (OCR, document parsing, semantic validation, PDF export, and multilingual translation).

---

## 📂 Monorepo Structure

```text
├── apps
│   ├── api            # Go Core API Gateway
│   │   ├── cmd        # Entrypoint (main.go)
│   │   └── internal   # Route Handlers, Clerk JWT Middleware, Convex HTTP Client
│   ├── ai-worker      # Python FastAPI Microservice (OCR, Translation, PDF Export)
│   └── web            # TanStack Start Frontend (Vite, GSAP, WebGL Voronoi, Clerk Integration)
├── convex             # Convex Database Schemas, Mutations, Actions, and Queries
├── docker-compose.yml # Dev orchestration config
├── test_apis.sh       # E2E API Verification Script
└── README.md          # Project documentation
```

---

## 🛠 Tech Stack

### Frontend Web App (`apps/web`)
- **TanStack Start** (React router & SSR framework)
- **Clerk React** (User authentication with custom `/sign-in` & `/sign-up` routes)
- **GSAP** (Smooth scroll-down and stagger reveal animations)
- **WebGL / GLSL Shaders** (Interactive, pointer-responsive Voronoi Cells background)
- **Vanilla CSS** (Dark-emerald-mint professional design token system)

### Go API Gateway (`apps/api`)
- **Go 1.25**
- **go-chi/chi** (Router & Middleware)
- **golang-jwt/jwt** (RS256 JWKS-based JWT verification)

### Python AI Worker (`apps/ai-worker`)
- **FastAPI** & **Uvicorn**
- **PyMuPDF**, **pdfplumber**, **python-docx** (Document processing)
- **fpdf2** & **Jinja2** (PDF generation)

### Persistence & Real-time Sync
- **Convex** (Serverless Database, mutations, query engine)

---

## 🚀 Getting Started

### 1. Prerequisites
Ensure you have the following installed locally:
- [Node.js](https://nodejs.org/) (v18+)
- [Go](https://go.dev/doc/install) (v1.25+)
- [Python 3.10+](https://www.python.org/downloads/)
- [Docker & Docker Compose](https://docs.docker.com/get-docker/)
- [Convex CLI](https://docs.convex.dev/cli)

### 2. Environment Setup

Create a `.env` file in the root directory based on the `.env.example` template:

```env
# Clerk Identity Verification
CLERK_SECRET_KEY=sk_test_...
CLERK_JWT_ISSUER=https://<your-app>.clerk.accounts.dev
CLERK_JWKS_URL=https://<your-app>.clerk.accounts.dev/.well-known/jwks.json
VITE_CLERK_PUBLISHABLE_KEY=pk_test_...

# Convex Deployment
CONVEX_URL=https://<your-deployment>.convex.cloud
```

Create `.env.local` inside the root and configure Convex URL settings matching your local server context if deploying database functions locally.

### 3. Deploying Convex Functions
To initialize the database schema and upload queries/mutations to your deployment:
```bash
npx convex dev
```

### 4. Running the Web Application
Run the root development script to boot the frontend Vite dev server (port `3000` / `3001`):
```bash
npm run dev
```

To compile and verify the production build for both client and server targets:
```bash
npm run build
```

### 5. Running Backends locally via Docker Compose
To start the backing microservices (Go API, Python AI-Worker, Redis, MinIO) in the background:
```bash
docker-compose up -d --build
```
This boots:
- **Go Core API** on `http://localhost:8080`
- **FastAPI AI Worker** on `http://localhost:8000`

---

## 🧪 Verification & Testing

Verify that the local services are healthy by running the automated End-to-End verification script:
```bash
chmod +x test_apis.sh
./test_apis.sh
```

This verification script walks through:
1. Simulating Clerk-authenticated login flows.
2. Checking claim dispute eligibility.
3. Creating cases and recording consents.
4. Reserving documents and uploading PDF evidence.
5. Triggering Python-based OCR/RAG reasoning processes.
6. Pulling extracted citations and generating PDF representations.

---

## 🛡 Security & Hardening Features

- **Identity Isolation:** All user profiles sync dynamically into Convex. Role-based Access Control (RBAC) restricts policyholder visibility to owned records, while review modules are limited to `REVIEWER`, `SENIOR_REVIEWER`, and `ADMIN` scopes.
- **JWKS Cache Caching:** Clerk RS256 token verification features a thread-safe local JWKS public-key cache with automatic rate-limited refreshing, avoiding external roundtrips.
- **Request Boundaries:** Global middleware locks incoming HTTP body size payload limits to **10MB** to prevent Denial of Service (DoS) attacks.
- **CORS Allowlisting:** Wildcard CORS is deprecated in favor of a strict environment-based origin validation.