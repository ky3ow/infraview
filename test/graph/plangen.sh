#!/usr/bin/env sh

if [ "$#" -ne 1 ]; then
  printf "Usage: $0 <plandir>"
  exit 1
fi

plandir="$1"
shift

mkdir -p "$plandir"

cd "$plandir"

terraform init
terraform plan -out "./plan.out"
terraform show -json "./plan.out" | jq > "./plan.json"

terraform apply "./plan.out"
rm "./plan.out"
