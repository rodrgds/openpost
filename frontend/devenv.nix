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
      cd "${config.git.root}/web"
      bun install --frozen-lockfile
      bun run format
    '';
  };
  eslint-wrapper = pkgs.writeShellApplication {
    name = "eslint-wrapper";
    runtimeInputs = [ pkgs.bun ];
    text = ''
      cd "${config.git.root}/web"
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
    web-dev.exec = ''
      cd web && bun install && bun run dev
    '';

    web-build.exec = ''
      cd web && bun install && bun run build
    '';

    web-test.exec = ''
      cd web && bun run test
    '';

    web-check.exec = ''
      cd web && bun run check
    '';

    web-lint.exec = ''
      cd web && bun run lint
    '';

    web-format.exec = ''
      cd web && bun run format
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
