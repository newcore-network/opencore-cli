#!/usr/bin/env bash

set -euo pipefail

build_dir=${1:-build}

copy_binary() {
  local package_dir=$1
  local source_binary=$2
  local package_binary=$3

  mkdir -p "npm/${package_dir}/bin"
  cp "${build_dir}/${source_binary}" "npm/${package_dir}/bin/${package_binary}"
  if [[ "${package_binary}" != *.exe ]]; then
    chmod 755 "npm/${package_dir}/bin/${package_binary}"
  fi
}

copy_binary linux-x64 opencore-linux-amd64 opencore
copy_binary linux-arm64 opencore-linux-arm64 opencore
copy_binary darwin-x64 opencore-darwin-amd64 opencore
copy_binary darwin-arm64 opencore-darwin-arm64 opencore
copy_binary win32-x64 opencore-windows-amd64.exe opencore.exe
