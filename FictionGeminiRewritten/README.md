# FictionGeminiRewritten

This project is a rewritten version of FictionGemini, an interactive story generation tool using Google's Gemini AI. It features a Go backend and a React frontend.

## Prerequisites

Before you begin, ensure you have the following installed:

*   **Go:** For running the backend server. (Version 1.20+ recommended)
*   **Node.js and npm:** For running the frontend React application. (Node.js LTS version recommended)
*   **Git:** For cloning the repository (if you haven't already).

## Backend Setup and Execution

The backend server is written in Go and handles the story generation logic by interacting with the Gemini API.

1.  **Navigate to the project directory:**
    ```bash
    cd /path/to/FictionGeminiRewritten
    ```

2.  **Build the server executable:**
    From the `/path/to/FictionGeminiRewritten` directory:
    ```bash
    go build ./cmd/server
    ```
    This will create an executable named `server` (or `server.exe` on Windows) in the `FictionGeminiRewritten` directory.

3.  **Run the backend server:**
    ```bash
    ./server
    ```
    By default, the server will start on port `8080`. If you need to use a different port, set the `PORT` environment variable:
    ```bash
    PORT=51605 ./server
    ```
    The server listens for requests on the `/generate` endpoint. Logs and generated story segments (in JSON format) are stored in the `jsons/` directory (created automatically).

## Frontend Setup and Execution

The frontend is a React application built with Vite.

1.  **Navigate to the frontend directory:**
    ```bash
    cd /path/to/FictionGeminiRewritten/frontend
    ```

2.  **Install dependencies:**
    If you haven't already, install the necessary Node.js packages:
    ```bash
    npm install
    ```

3.  **Run the frontend development server:**
    ```bash
    npm run dev
    ```
    This will typically start the frontend application on `http://localhost:5173` (Vite's default) or another available port if 5173 is busy. The console output from `npm run dev` will show the exact URL.
    *Note: The application was developed targeting frontend port `56614` and backend port `51605`. The frontend `App.jsx` is currently hardcoded to connect to the backend at `http://localhost:51605/generate`. If your backend runs on a different port, you'll need to update this in `/frontend/src/App.jsx`.*

## API Key

The application requires a Google Gemini API key to function.
*   When you first load the frontend application in your browser, you will be prompted to enter your API key.
*   This key is then stored in your browser's `localStorage` for subsequent sessions.
*   The API key is sent to the backend with each request to authenticate with the Gemini API.

## Project Structure

*   `/cmd/server/main.go`: Main entry point for the backend server.
*   `/internal/`: Contains the core backend logic (AI interaction, services, handlers).
*   `/frontend/`: Contains the React frontend application.
    *   `/frontend/src/App.jsx`: Main React component for the frontend UI and logic.
*   `/jsons/`: Directory where backend logs and story outputs are saved (auto-created).
*   `server` (or `server.exe`): Compiled backend executable.
