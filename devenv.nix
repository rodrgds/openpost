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
    pkgs.make
    pkgs.wget
  ];

  # Environment variables
  env.OPENPOST_PORT = "8080";
  env.OPENPOST_DB_PATH = "file:openpost.db?cache=shared&mode=rwc";
  env.OPENPOST_FRONTEND_URL = "http://localhost:8080";

  # Scripts available in the shell
  scripts = {
    dev.exec = ''
      make dev
    '';

    build.exec = ''
      make build
    '';

    test-all.exec = ''
      make test
    '';

    clean.exec = ''
      make clean
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
    echo "  Available commands:"
    echo "    dev       - Start frontend and backend dev servers"
    echo "    build     - Build production binary"
    echo "    test-all  - Run all tests"
    echo "    clean     - Clean build artifacts"
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
