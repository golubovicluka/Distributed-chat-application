---

# Distributed Chat Application

A real-time distributed chat application built with Go backend services and React frontend, featuring load balancing, WebSocket communication, and message persistence.

## 📐 Architecture

This application consists of three main components:

### Load Balancer (`loadbalancer/`)

* **Port**: 9000
* **Purpose**: Manages multiple chat server instances using least-connection routing
* **Features**:

  * REST API for server registration and load reporting
  * Thread-safe server state management with mutex protection
  * Automatic server health monitoring

### Chat Server (`server/`)

* **Technology**: Go with WebSocket support
* **Features**:

  * Configurable host/port (default: 127.0.0.1:8080)
  * SQLite database for message persistence
  * File logging to `chat.log`
  * Auto-registration with load balancer
  * Real-time client load reporting
  * Concurrent client handling with goroutines

### Chat Frontend (`chat-app/`)

* **Technology**: React 19 + TypeScript + Vite
* **UI Library**: shadcn/ui with Tailwind CSS v4
* **Features**:

  * Real-time messaging interface
  * User authentication
  * Chat history retrieval
  * Responsive design with modern UI components

## 🗄 Database Schema

```sql
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT,
    message TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
)
```

## ⚙ Installation & Setup

### Prerequisites

* **Go**: Version 1.24.5 or higher
* **Node.js**: Version 18 or higher
* **npm**: Latest version
* **Docker**: For running Redis container

### 1. Clone the Repository

```bash
git clone <repository-url>
cd into project
```

### 2. Start Redis with Docker

Start a Redis container:

```bash
docker run -d --name redis-chat -p 6379:6379 redis:latest
```

### 3. Backend Setup

#### Install Go Dependencies

Navigate to the server directory:

```bash
cd server
go mod tidy
```

Navigate to the loadbalancer directory:

```bash
cd ../loadbalancer
go mod tidy
```

### 4. Frontend Setup

Navigate to the chat-app directory:

```bash
cd ../chat-app
npm install
```

## ▶ Running the Application

### Step 1: Ensure Redis is Running

Make sure your Redis container is running:

```bash
docker ps | grep redis-chat
```

If not running, start it:

```bash
docker start redis-chat
```

### Step 2: Start the Load Balancer

```bash
cd loadbalancer
go run main.go
```

The load balancer will start on port 9000.

### Step 3: Start Chat Server(s)

In a new terminal:

```bash
cd server
go run main.go -host 127.0.0.1 -port 8080
```

You can start multiple chat servers on different ports:

```bash
go run main.go -host 127.0.0.1 -port 8081
go run main.go -host 127.0.0.1 -port 8082
```

### Step 4: Start the Frontend

In a new terminal:

```bash
cd chat-app
npm run dev
```

The React application will start on [http://localhost:5173](http://localhost:5173)

## 🛠 Development Commands

### Frontend (chat-app/)

```bash
npm run dev      # Start development server with hot reload
npm run build    # Build for production
npm run lint     # Run ESLint linter
npm run preview  # Preview production build
```

### Backend (server/ and loadbalancer/)

```bash
go run main.go   # Run the service directly
go build         # Compile the binary
go mod tidy      # Clean up dependencies
```

## 🌐 API Endpoints

### Load Balancer (Port 9000)

* `POST /register` - Register a new chat server
* `POST /update` - Update server load information
* `GET /get` - Get optimal server for client connection

### Chat Server

* `GET /ws?username=<name>` - WebSocket endpoint for chat connections

## 🔄 Communication Flow

1. **Server Registration**: Chat servers register with the load balancer on startup
2. **Load Reporting**: Servers continuously report their client load to the load balancer
3. **Server Selection**: Frontend queries load balancer to get the optimal server
4. **Connection**: WebSocket connections are established between frontend and selected chat server
5. **Messaging**: Messages are persisted to SQLite database and broadcast to all connected clients

## 📦 Key Dependencies

### Backend

* `gorilla/websocket` - WebSocket handling
* `mattn/go-sqlite3` - SQLite database driver
* `go-redis/redis/v8` - Redis integration

### Frontend

* `react` v19.1.1 - UI framework
* `typescript` - Type safety
* `vite` - Build tool and dev server
* `tailwindcss` v4.1.11 - Styling
* `@radix-ui/react-slot` - UI primitives
* `lucide-react` - Icons
* `eslint` - Code linting
