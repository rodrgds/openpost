{ pkgs, ... }:

{
  # Go language support
  languages.go = {
    enable = true;
    package = pkgs.go_1_24;
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
      cd backend && go test ./...
    '';

    backend-lint.exec = ''
      cd backend && golangci-lint run
    '';
  };

  # Git hooks
  git-hooks.hooks = {
    gofmt.enable = true;
  };
}
