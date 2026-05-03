{
  pkgs,
  lib,
  config,
  ...
}:

{
  # Shared packages for the entire project
  packages = [
    pkgs.git
    pkgs.curl
    pkgs.jq
    pkgs.ripgrep
    pkgs.sqlite
    pkgs.wget
    pkgs.docker
    pkgs.golangci-lint
  ];

  # Environment variables
  env.OPENPOST_PORT = "8080";
  env.OPENPOST_DATABASE_PATH = "file:openpost.db?cache=shared&mode=rwc";
  env.OPENPOST_APP_URL = "http://localhost:8080";

  # Scripts available in the shell
  scripts = {
    app.exec = ''
      # Start frontend dev server directly — exec replaces the subshell with bun,
      # so $! is bun's PID directly (no wrapper shell process in between).
      (cd frontend && bun install && exec bun run dev) &
      FRONTEND_PID=$!

      cleanup() {
        # SIGKILL bun/vite directly. No wrapper shell, no process groups, no wait.
        kill -9 $FRONTEND_PID 2>/dev/null || true
      }
      trap cleanup EXIT

      backend-run
    '';

    docs.exec = ''
      cd docs-site
      bun install
      bun run docs:dev
    '';

    dev.exec = ''
      (cd frontend && bun install && exec bun run dev) &
      FRONTEND_PID=$!

      (cd docs-site && bun install && exec bun run docs:dev) &
      DOCS_PID=$!

      cleanup() {
        kill -9 $FRONTEND_PID 2>/dev/null || true
        kill -9 $DOCS_PID 2>/dev/null || true
      }
      trap cleanup EXIT

      backend-run
    '';

    build.exec = ''
      frontend-build && backend-build
    '';

    docs-build.exec = ''
      cd docs-site
      bun install
      bun run docs:build
    '';

    test-all.exec = ''
      backend-test && frontend-test
    '';

    lint.exec = ''
      frontend-lint
      frontend-check
      backend-format-check
      backend-lint
    '';

    clean.exec = ''
      rm -rf backend/openpost
      rm -rf frontend/.svelte-kit
      rm -rf frontend/node_modules
      rm -f backend/*.db
    '';

    install.exec = ''
      (cd frontend && bun install)
      (cd docs-site && bun install)
      (cd backend && go mod download)
    '';

    setup.exec = ''
      cp backend/.env.example backend/.env
      echo "Created backend/.env - edit with your OAuth credentials"
    '';

    docker-build.exec = ''
      docker build -t openpost:latest -f docker/Dockerfile .
    '';

    docker-run.exec = ''
      docker run -d -p 8080:8080 --name openpost openpost:latest
    '';
  };

  # Shell initialization
  enterShell = ''
    echo ""
    echo "  OpenPost Development Environment"
    echo "  --------------------------------"
    echo "  Go:     $(go version 2>/dev/null || echo 'not installed')"
    echo "  Bun:    $(bun --version 2>/dev/null || echo 'not installed')"
    echo ""
    echo "  Commands:"
    echo "    app          - Start frontend and backend dev servers"
    echo "    docs         - Start the VitePress docs site"
    echo "    dev          - Start frontend, backend, and docs together"
    echo "    build        - Build production binary"
    echo "    docs-build   - Build the VitePress docs site"
    echo "    test-all     - Run all tests"
    echo "    clean        - Clean build artifacts"
    echo "    install      - Install frontend, docs, and backend dependencies"
    echo "    setup        - Create .env from example"
    echo "    docker-build - Build Docker image"
    echo "    docker-run   - Run Docker container"
    echo ""

    # Load .env if it exists
    if [ -f backend/.env ]; then
      set -a
      source backend/.env
      set +a
    fi
  '';

  # Test that key tools are available
  enterTest = ''
    go version
    bun --version
    git --version
  '';
}
