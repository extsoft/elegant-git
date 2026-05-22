#!/usr/bin/env bash
# Usage:
#   ./install.bash                           installs to "${HOME}/.elegant-git" from remote git
#   ./install.bash /installation/path        installs to "/installation/path" from remote git
#   ./install.bash /installation/path src    installs to "installation/path" from local sources
set -e

REPO_HOME="https://github.com/bees-hive/elegant-git.git"

copy() {
  local FROM=${1}
  local INTO=${2}
  install -d -m 755 ${INTO}/{bin,libexec,completions}
  install -d -m 755 ${INTO}/libexec/plugins
  install -m 755 ${FROM}/bin/git* ${INTO}/bin
  install -m 755 ${FROM}/libexec/git* ${INTO}/libexec
  install -m 755 ${FROM}/libexec/plugins/* ${INTO}/libexec/plugins
  install -m 644 ${FROM}/completions/* ${INTO}/completions
  install -m 644 ${FROM}/LICENSE ${INTO}
  install -m 644 ${FROM}/README.md ${INTO}
  install -m 644 ${FROM}/version ${INTO}
}

update-path() {
  local INSTALL_PATH=${1}
  cat <<TEXT
You need to add Elegant Git to the PATH by adding the folowing line
to relevant shell configuration file ('~/.bashrc', '~/.bash_profile',
'~/.zshrc', etc.):

    export PATH=${INSTALL_PATH}/bin:\$PATH

After, please configure the installation by running

    git-elegant acquire-git

TEXT
}

update-completion() {
  local INSTALL_PATH=${1}
  local BASH_COMPLETION="${INSTALL_PATH}/completions/git-elegant.bash"
  local ZSH_COMPLETION="${INSTALL_PATH}/completions/_git-elegant"
  cat <<TEXT

Completion installation
=======================

Bash competion
--------------
You need to load a Bash completion file from a relevant shell
configuration file ('~/.bashrc', '~/.bash_profile', etc.) by
adding the following line:

    [ -f ${BASH_COMPLETION} ] && source ${BASH_COMPLETION}

Elegant Git's Bash completion does not work without regular Git
completion. If you don't have it, please install it following
https://github.com/bobthecow/git-flow-completion/wiki/Install-Bash-git-completion.

Zsh completion
--------------
1. link completion file into a completion directory

    mkdir -p ~/.zsh/completion
    ln -s ${ZSH_COMPLETION} ~/.zsh/completion/

2. include the completion directory in your '\${fpath}'

    fpath=(~/.zsh/completion \${fpath})

3. make sure compinit is loaded or do it by adding in '~/.zshrc:'

    autoload -Uz compinit && compinit -i

4. (optional) consider using
https://raw.githubusercontent.com/zsh-users/zsh/master/Completion/Unix/Command/_git
as Git completion file that provides great completions for Git commands and plugins
Elegant Git's completion file.

    curl -L https://raw.githubusercontent.com/zsh-users/zsh/master/Completion/Unix/Command/_git \\
        > ~/.zsh/completion/_git


P.S.
----
Please restart terminal session in order to activate completion.

TEXT
}

add-version() {
  cd "${1}"
  git describe >version
  cd -
}

main() {
  if [[ -n ${1} ]]; then
    INSTALL_PATH="${1}"
    shift
  fi
  : ${INSTALL_PATH:="${HOME}/.elegant-git"}
  # mode selection
  if [[ -z ${1} ]]; then
    local CODE="/tmp/elegant-git"
    git clone --quiet --depth 50 ${REPO_HOME} ${CODE}
    add-version ${CODE}
    copy ${CODE} ${INSTALL_PATH}
    rm -r ${CODE}
  else
    local path="${0%/*}"
    add-version "${path}"
    copy "${path}" ${INSTALL_PATH}
  fi
  echo "Elegant Git is installed to '${INSTALL_PATH}/bin/git-elegant'."
  if command -v git-elegant 1>/dev/null 2>&1; then
    git-elegant acquire-git
  else
    update-path ${INSTALL_PATH}
  fi
  update-completion ${INSTALL_PATH}
}

main $@
