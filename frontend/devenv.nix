{
  config,
  pkgs,
  lib,
  ...
}:
let
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
      bun install --frozen-lockfile
      bun run check
    '';
  };
  vitest-wrapper = pkgs.writeShellApplication {
    name = "vitest-wrapper";
    runtimeInputs = [ pkgs.bun ];
    text = ''
      cd "${config.git.root}/frontend"
      bun install --frozen-lockfile
      # Run tests only if test files exist, otherwise skip silently
      if find src -name "*.test.ts" -o -name "*.spec.ts" 2>/dev/null | grep -q .; then
        bun run test
      else
        echo "No test files found, skipping tests..."
        exit 0
      fi
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
      ${lib.getExe vitest-wrapper}
    '';

    frontend-check.exec = ''
      ${lib.getExe svelte-check-wrapper}
    '';

    frontend-lint.exec = ''
      ${lib.getExe eslint-wrapper}
    '';

    frontend-format.exec = ''
      cd frontend && bun run format
    '';
  };

  # Git hooks - all must pass to allow commits
  git-hooks.hooks = {
    # Lint check (prettier + eslint)
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
