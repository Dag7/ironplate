# Enable prompt command substitution
setopt PROMPT_SUBST

# K8s context with short name and color
# Colors: prod=red, staging=yellow, k3d=blue
function k8s_prompt() {
  local ctx=$(kubectl config current-context 2>/dev/null)
  local name color
  case "$ctx" in
    *production*|*prod*) name="PROD"; color="%{$fg[red]%}" ;;
    *staging*)           name="STG"; color="%{$fg[yellow]%}" ;;
    k3d-*)               name="K3D"; color="%{$fg[blue]%}" ;;
    "")                  name="-"; color="%{$fg[white]%}" ;;
    *)                   name="${ctx:0:10}"; color="%{$fg[white]%}" ;;
  esac
  echo "${color}[${name}]%{$reset_color%}"
}
