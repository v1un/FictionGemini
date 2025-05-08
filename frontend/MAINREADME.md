# AI Fiction Forge - Frontend

This directory contains the React frontend for the AI Fiction Forge application.

## Prerequisites

- Node.js (which includes npm) installed on your system. You can download it from [nodejs.org](https://nodejs.org/).
- The Go backend server (`FictionGemini/main.go`) should be running, typically on `http://localhost:8080`.

## Setup and Running

1.  **Navigate to the frontend directory:**
    ```bash
    cd FictionGemini/frontend
    ```

2.  **Install dependencies:**
    If you're using npm:
    ```bash
    npm install
    ```
    If you're using yarn:
    ```bash
    yarn install
    ```

3.  **Start the React development server:**
    If you're using npm:
    ```bash
    npm start
    ```
    If you're using yarn:
    ```bash
    yarn start
    ```

This will typically open the application in your default web browser at `http://localhost:3000`.

The React app is configured to proxy API requests to `/generate` to `http://localhost:8080/generate` (where the Go backend is expected to be running). This is handled by the `"proxy": "http://localhost:8080"` line in `package.json`.

## Available Scripts

In the project directory, you can run:

### `npm start` or `yarn start`

Runs the app in development mode.
Open [http://localhost:3000](http://localhost:3000) to view it in the browser.

The page will reload if you make edits.
You will also see any lint errors in the console.

### `npm test` or `yarn test`

Launches the test runner in interactive watch mode.
See the section about [running tests](https://facebook.github.io/create-react-app/docs/running-tests) for more information.

### `npm run build` or `yarn build`

Builds the app for production to the `build` folder.
It correctly bundles React in production mode and optimizes the build for the best performance.

The build is minified and the filenames include the hashes.
Your app is ready to be deployed!

See the section about [deployment](https://facebook.github.io/create-react-app/docs/deployment) for more information.

### `npm run eject` or `yarn eject`

**Note: this is a one-way operation. Once you `eject`, you can’t go back!**

If you aren’t satisfied with the build tool and configuration choices, you can `eject` at any time. This command will remove the single build dependency from your project.

Instead, it will copy all the configuration files and the transitive dependencies (webpack, Babel, ESLint, etc) right into your project so you have full control over them. All of the commands except `eject` will still work, but they will point to the copied scripts so you can tweak them. At this point you’re on your own.

You don’t have to ever use `eject`. The curated feature set is suitable for small and middle deployments, and you shouldn’t feel obligated to use this feature. However, we understand that this tool wouldn’t be useful if you couldn’t customize it when you are ready for it.