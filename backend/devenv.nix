{
  pkgs,
  lib,
  ...
}:

let
  backend-gofmt-check = pkgs.writeShellApplication {
    name = "backend-gofmt-check";
    text = ''
      cd backend
      unformatted=$(gofmt -l .)
      if [ -n "$unformatted" ]; then
        echo "$unformatted"
        exit 1
      fi
    '';
  };

  backend-golangci-lint = pkgs.writeShellApplication {
    name = "backend-golangci-lint";
    text = ''
      mkdir -p backend/cmd/openpost/public
      touch backend/cmd/openpost/public/.gitkeep
      cd backend
      golangci-lint run ./...
    '';
  };

  backend-go-test = pkgs.writeShellApplication {
    name = "backend-go-test";
    text = ''
      mkdir -p backend/cmd/openpost/public
      touch backend/cmd/openpost/public/.gitkeep
      cd backend
      go test ./...
    '';
  };
in

{
  # Go language support
  languages.go = {
    enable = true;
    package = pkgs.go_1_25;
  };

  # Additional packages for backend development
  packages = [
    pkgs.golangci-lint
    pkgs.gotools
    pkgs.sqlc
  ];

  # Scripts for backend development
  scripts = {
    backend-run.exec = ''
      cd backend && go run ./cmd/openpost
    '';

    backend-build.exec = ''
      cd backend && go build -o openpost ./cmd/openpost
    '';

    backend-test.exec = ''
      ${lib.getExe backend-go-test}
    '';

    backend-format-check.exec = ''
      ${lib.getExe backend-gofmt-check}
    '';

    backend-lint.exec = ''
      ${lib.getExe backend-golangci-lint}
    '';
  };

  # Git hooks - all must pass to allow commits
  git-hooks.hooks = {
    # Format check (go fmt)
    gofmt = {
      enable = true;
      entry = lib.getExe backend-gofmt-check;
      pass_filenames = false;
    };

    # Lint check (golangci-lint)
    golangci-lint = {
      enable = true;
      entry = lib.getExe backend-golangci-lint;
      files = "\\.go$";
      pass_filenames = false;
    };

    # Unit tests (go test)
    go-test = {
      enable = true;
      entry = lib.getExe backend-go-test;
      files = "\\.go$";
      pass_filenames = false;
    };
  };
}
