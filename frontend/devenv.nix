{
  config,
  pkgs,
  lib,
  ...
}:
let
  npm-format = pkgs.writeShellApplication {
    name = "npm-format";
    runtimeInputs = [ pkgs.bun ];
    text = ''
      cd "${config.git.root}/frontend"
      bun install --frozen-lockfile
      bun run format
    '';
  };
  eslint-wrapper = pkgs.writeShellApplication {
    name = "eslint-wrapper";
    runtimeInputs = [ pkgs.bun ];
    text = ''
      cd "${config.git.root}/frontend"
      bun install --frozen-lockfile
      bun run lint
    '';
  };
  svelte-check-wrapper = pkgs.writeShellApplication {
    name = "svelte-check-wrapper";
    runtimeInputs = [ pkgs.bun ];
    text = ''
      cd "${config.git.root}/frontend"
      bun run check
    '';
  };
  vitest-wrapper = pkgs.writeShellApplication {
    name = "vitest-wrapper";
    runtimeInputs = [ pkgs.bun ];
    text = ''
      cd "${config.git.root}/frontend"
      bun run test
    '';
  };
in
{
  # Bun language support
  languages.javascript = {
    enable = true;
    bun.enable = true;
  };

  # Scripts for frontend development
  scripts = {
    frontend-dev.exec = ''
      cd frontend && bun install && bun run dev
    '';

    frontend-build.exec = ''
      cd frontend && bun install && bun run build
    '';

    frontend-test.exec = ''
      cd frontend && bun run test
    '';

    frontend-check.exec = ''
      cd frontend && bun run check
    '';

    frontend-lint.exec = ''
      cd frontend && bun run lint
    '';

    frontend-format.exec = ''
      cd frontend && bun run format
    '';
  };

  # Git hooks - all must pass to allow commits
  git-hooks.hooks = {
    # Format check (prettier)
    npm-format = {
      enable = true;
      name = "prettier-npm";
      entry = "${lib.getExe npm-format}";
      files = "\\.(js|ts|svelte|css|html)$";
      pass_filenames = false;
    };

    # Lint check (eslint)
    eslint = {
      enable = true;
      entry = "${lib.getExe eslint-wrapper}";
      files = "\\.(js|ts|svelte)$";
      pass_filenames = false;
    };

    # Type check (svelte-check)
    svelte-check = {
      enable = true;
      entry = "${lib.getExe svelte-check-wrapper}";
      files = "\\.(ts|svelte)$";
      pass_filenames = false;
    };

    # Unit tests (vitest)
    vitest = {
      enable = true;
      entry = "${lib.getExe vitest-wrapper}";
      files = "\\.(ts|svelte)$";
      pass_filenames = false;
    };
  };
}
