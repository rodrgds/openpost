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
      bunx eslint .
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

  # Git hooks
  git-hooks.hooks = {
    eslint = {
      enable = true;
      entry = "${lib.getExe eslint-wrapper}";
      files = "\\.(js|ts|svelte)$";
      pass_filenames = false;
    };
    npm-format = {
      enable = true;
      name = "prettier-npm";
      entry = "${lib.getExe npm-format}";
      files = "\\.(js|ts|svelte|css|html)$";
      pass_filenames = false;
    };
  };
}
