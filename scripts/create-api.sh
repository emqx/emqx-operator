#!/usr/bin/env bash
set -euo pipefail
set -x

version=$1
kind=$2

kubebuilder create api --group apps  --version ${version} --kind ${kind} 

kubebuilder create webhook --group apps --version ${version} --kind ${kind} --defaulting --programmatic-validation --conversion