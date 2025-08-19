# Distributed Chat Application

A real-time distributed chat application built with Go backend services and React frontend, featuring load balancing, WebSocket communication, Redis pub/sub messaging, and SQLite persistence.

## Architecture

This application consists of three main components working together to provide scalable real-time messaging:

### Load Balancer (`loadbalancer/`)

- **Port**: 9000
- **Purpose**: Manages multiple chat server instances using least-connection routing
- **Features**:
  - REST API for server registration and load reporting (`/register`, `/update`, `/get`)
  - Thread-safe server state management with mutex protection
  - CORS middleware for browser compatibility
  - Automatic selection of optimal server based on current load

### Chat Server (`server/`)

- **Technology**: Go with WebSocket support and Redis pub/sub
- **Architecture**: Modular package structure following Go best practices
- **Features**:
  - Configurable host/port (default: 127.0.0.1:8080)
  - SQLite database for message persistence
  - Redis integration for real-time message broadcasting across server instances
  - Auto-registration with load balancer on startup
  - Continuous client load reporting to load balancer
  - WebSocket endpoint with ping/pong health checks
  - Chat history API endpoint (`/history`)
  - CORS middleware for cross-origin requests
- **Benefits of Refactored Architecture**:
  - **Maintainability**: Easy to locate and modify specific functionality
  - **Testability**: Individual packages can be tested in isolation
  - **Team Development**: Multiple developers can work on different modules
  - **Code Clarity**: Each package has a single, clear responsibility
  - **Go Best Practices**: Proper package organization with clean module imports

### Chat Frontend (`chat-app/`)

- **Technology**: React 19 + TypeScript + Vite
- **UI Library**: shadcn/ui with Tailwind CSS v4
- **Features**:
  - Real-time messaging interface with WebSocket connections
  - Automatic server selection via load balancer
  - Chat history retrieval and display
  - User authentication with username input
  - Responsive design with modern UI components (Cards, Buttons, Input fields)

## Database Schema

The database schema is defined in `server/database/db.go`:

```sql
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT,
    message TEXT,
    server TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
)
```

## Installation & Setup

### Prerequisites

- **Go**: Version 1.24.5 or higher
- **Node.js**: Version 18 or higher
- **npm**: Latest version
- **Redis**: For message broadcasting (Docker recommended)

### 1. Clone the Repository

```bash
git clone <repository-url>
cd CS230-PZ-LukaGolubovic6356
```

### 2. Start Redis with Docker

Start a Redis container for message broadcasting between chat server instances:

```bash
docker run -d --name redis-chat -p 6379:6379 redis:latest
```

### 3. Backend Setup

#### Install Go Dependencies

Navigate to the server directory and install dependencies:

```bash
cd server
go mod tidy
```

Note: The load balancer (`loadbalancer/`) is a standalone Go file without its own module.

### 4. Frontend Setup

Navigate to the chat-app directory and install dependencies:

```bash
cd chat-app
npm install
```

## Running the Application

Follow these steps in order to start all components:

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

The load balancer will start on port 9000 and display:

```
[LB] Load Balancer is running on :9000
```

### Step 3: Start Chat Server(s)

In a new terminal, start at least one chat server:

```bash
cd server
go run main.go -host 127.0.0.1 -port 8080
```

You can start multiple chat servers on different ports for load balancing:

```bash
# Terminal 2
go run main.go -host 127.0.0.1 -port 8081

# Terminal 3
go run main.go -host 127.0.0.1 -port 8082
```

Each server will:

- Create a SQLite database (`chat.db`) if it doesn't exist
- Register itself with the load balancer
- Start reporting its client load continuously

### Step 4: Start the Frontend

In a new terminal:

```bash
cd chat-app
npm run dev
```

The React application will start on http://localhost:5173

## Development Commands

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
go mod tidy      # Clean up dependencies (server only)
```

## API Endpoints

### Load Balancer (Port 9000)

- `POST /register` - Register a new chat server with the load balancer
- `POST /update` - Update server load information
- `GET /get` - Get optimal server for client connection based on current loads

### Chat Server

- `GET /ws?username=<name>` - WebSocket endpoint for real-time chat connections
- `GET /history` - REST endpoint to retrieve chat message history

## Communication Flow

1. **Server Registration**: Chat servers register with the load balancer on startup
2. **Load Reporting**: Servers continuously report their client count to the load balancer
3. **Server Selection**: Frontend queries load balancer (`/get`) to get the optimal server
4. **WebSocket Connection**: Client establishes WebSocket connection to selected server
5. **Message Broadcasting**: Messages are published to Redis and broadcast to all server instances
6. **Persistence**: Messages are stored in SQLite database for history retrieval

## Key Dependencies

### Backend

- **Go Modules**:
  - `github.com/gorilla/websocket` v1.5.3 - WebSocket handling
  - `github.com/mattn/go-sqlite3` v1.14.30 - SQLite database driver
  - `github.com/go-redis/redis/v8` v8.11.5 - Redis client for pub/sub messaging

### Frontend

- **React Ecosystem**:
  - `react` v19.1.1 - UI framework
  - `react-dom` v19.1.1 - DOM rendering
  - `typescript` ~5.8.3 - Type safety
  - `vite` v7.1.0 - Build tool and dev server
- **UI & Styling**:
  - `tailwindcss` v4.1.11 - Utility-first CSS framework
  - `@radix-ui/react-slot` v1.2.3 - UI primitives
  - `lucide-react` v0.539.0 - Icon library
  - `class-variance-authority` v0.7.1 - Component variant handling
- **Development**:
  - `eslint` v9.32.0 - Code linting
  - `@vitejs/plugin-react-swc` v3.11.0 - Fast refresh support

## Project Structure

```
CS230-PZ-LukaGolubovic6356/
├── chat-app/                 # React frontend application
│   ├── src/
│   │   ├── components/       # React components (ChatRoom, Login, UI)
│   │   ├── services/         # API and WebSocket services
│   │   └── lib/             # Utility functions
│   ├── package.json
│   └── vite.config.ts
├── server/                   # Go chat server with modular architecture
│   ├── main.go              # Application bootstrap and dependency wiring
│   ├── models/              # Data structures and types
│   │   └── message.go       # Message model definition
│   ├── database/            # Database operations and schema
│   │   └── db.go            # SQLite initialization and table creation
│   ├── client/              # WebSocket client management
│   │   └── client.go        # Client connection handling and message pumps
│   ├── hub/                 # Client connection hub and message broadcasting
│   │   └── hub.go           # Central message distribution and Redis pub/sub
│   ├── handlers/            # HTTP request handlers
│   │   ├── websocket.go     # WebSocket upgrade and connection handling
│   │   └── history.go       # Chat history API endpoint
│   ├── middleware/          # HTTP middleware
│   │   └── cors.go          # Cross-origin request handling
│   ├── loadbalancer/        # Load balancer communication
│   │   └── client.go        # Registration and load reporting
│   ├── go.mod               # Go module dependencies
│   ├── go.sum               # Dependency checksums
│   └── chat.db              # SQLite database (auto-generated)
├── loadbalancer/            # Go load balancer (standalone)
│   └── main.go              # Load balancer implementation
├── CLAUDE.md               # Development instructions for Claude Code
└── README.md              # This file
```

### Server Package Responsibilities

- **`main.go`** (56 lines): Application entry point, dependency injection, and HTTP server setup
- **`models/message.go`** (9 lines): Message data structure with JSON serialization tags
- **`database/db.go`** (32 lines): SQLite database initialization, schema creation, and table setup
- **`client/client.go`** (95 lines): WebSocket client lifecycle, read/write message pumps, and connection management
- **`hub/hub.go`** (108 lines): Central message hub, client registry, Redis pub/sub integration, and broadcast logic
- **`handlers/websocket.go`** (37 lines): WebSocket protocol upgrade and client authentication
- **`handlers/history.go`** (33 lines): REST API endpoint for retrieving chat message history
- **`middleware/cors.go`** (16 lines): Cross-origin resource sharing configuration for browser compatibility
- **`loadbalancer/client.go`** (40 lines): Load balancer registration and periodic load reporting

## Features

- **Real-time Messaging**: WebSocket connections with automatic reconnection
- **Load Balancing**: Intelligent server selection based on current client loads
- **Message Persistence**: SQLite database storage with chat history API
- **Scalability**: Redis pub/sub enables horizontal scaling of chat servers
- **Modern UI**: Responsive design with shadcn/ui components and Tailwind CSS
- **Modular Architecture**: Clean separation of concerns with single-responsibility packages
- **Developer Experience**: Hot reload, TypeScript support, comprehensive linting, and maintainable code structure
- **Health Monitoring**: WebSocket ping/pong mechanism and server load reporting
- **CORS Support**: Cross-origin resource sharing for browser compatibility
- **Code Quality**: Go best practices with testable, modular design for enterprise-grade development

## Troubleshooting

### Common Issues

1. **Redis connection failed**:

   - Ensure Docker is running: `docker --version`
   - Check Redis container status: `docker ps | grep redis-chat`
   - Start Redis if stopped: `docker start redis-chat`

2. **Port already in use**:

   - Check what's using the ports: `netstat -tulpn | grep :8080`
   - Make sure no other services are running on ports 6379 (Redis), 8080+ (servers), 9000 (load balancer), or 5173 (frontend)

3. **Database permissions**:

   - Ensure the server directory has write permissions for SQLite database files
   - Check file ownership: `ls -la server/chat.db`

4. **WebSocket connection fails**:

   - Verify the complete startup sequence: Redis → Load Balancer → Chat Server(s) → Frontend
   - Check browser developer console for connection errors
   - Verify load balancer can reach chat servers: `curl http://localhost:9000/get`

5. **Frontend build errors**:
   - Clear node modules: `rm -rf chat-app/node_modules && cd chat-app && npm install`
   - Check Node.js version: `node --version` (should be 18+)

### Logs and Debugging

- **Chat server logs**: Check `server/chat.log` for detailed server operations
- **Frontend logs**: Open browser developer console (F12) for client-side errors
- **Load balancer logs**: Terminal output shows registration and routing decisions
- **Redis logs**: `docker logs redis-chat` for Redis container logs

### Performance Tips

- Start multiple chat server instances on different ports for better load distribution
- Monitor server loads in load balancer logs to verify balanced distribution
- Use Redis persistence if message history across restarts is required
