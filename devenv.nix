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
  ];

  # Environment variables
  env.OPENPOST_PORT = "8080";
  env.OPENPOST_DB_PATH = "file:openpost.db?cache=shared&mode=rwc";
  env.OPENPOST_FRONTEND_URL = "http://localhost:8080";

  # Scripts available in the shell
  scripts = {
    dev.exec = ''
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

    build.exec = ''
      frontend-build && backend-build
    '';

    test-all.exec = ''
      backend-test && frontend-test
    '';

    clean.exec = ''
      rm -rf backend/openpost
      rm -rf frontend/.svelte-kit
      rm -rf frontend/node_modules
      rm -f backend/*.db
    '';

    install.exec = ''
      frontend-build
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
    echo "    dev          - Start frontend and backend dev servers"
    echo "    build        - Build production binary"
    echo "    test-all     - Run all tests"
    echo "    clean        - Clean build artifacts"
    echo "    install      - Install dependencies"
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
