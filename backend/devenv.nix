{
  pkgs,
  lib,
  ...
}:

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
      cd backend && go test ./...
    '';

    backend-lint.exec = ''
      cd backend && golangci-lint run
    '';
  };

  # Git hooks - all must pass to allow commits
  git-hooks.hooks = {
    # Format check (go fmt)
    gofmt = {
      enable = true;
    };

    # Lint check (golangci-lint)
    golangci-lint = {
      enable = true;
      entry = "${lib.getExe pkgs.golangci-lint}";
      files = "\\.go$";
      pass_filenames = false;
    };

    # Unit tests (go test)
    go-test = {
      enable = true;
      entry = lib.getExe (pkgs.writeShellApplication {
        name = "go-test";
        text = ''
          cd backend && go test ./...
        '';
      });
      files = "\\.go$";
      pass_filenames = false;
    };
  };
}
