# Nix Module

OpenPost can also be deployed through a NixOS module. This is the production setup behind `https://op.rgo.pt`.

## Source

- Live module source: [rodrgds/nix-config/modules/services/openpost/default.nix](https://github.com/rodrgds/nix-config/blob/main/modules/services/openpost/default.nix)
- Raw file used during docs builds: [raw.githubusercontent.com/.../default.nix](https://raw.githubusercontent.com/rodrgds/nix-config/refs/heads/main/modules/services/openpost/default.nix)

## What this example shows

- Running OpenPost as an OCI container
- Persisting SQLite and media storage under `/var/lib/openpost`
- Supplying secrets through `sops`
- Wiring public callback and media URLs to the deployed domain
- Exposing the service through your existing reverse proxy layer

## Current module

The snippet below is refreshed at docs build time from the source repository above. If the fetch fails, the docs fall back to the last generated copy committed in this repo.

<!--@include: ../.generated/openpost-nix-module.md-->
