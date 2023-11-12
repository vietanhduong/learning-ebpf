#!/bin/bash
set -e

TARGET_KERNEL="5.10.173"

REPO_ROOT=$(git rev-parse --show-toplevel)
BUILDS_DIR="${REPO_ROOT}/.builds"
SCRIPTS_DIR="${REPO_ROOT}/scripts"
KERNEL_IMAGE=${BUILDS_DIR}/bzImage
DISK_IMAGE=${BUILDS_DIR}/qemu-${TARGET_KERNEL}.qcow2
RUN_SSHD_PATH=$REPO_ROOT/scripts/run_sshd.sh
RUN_QEMU_SCRIPT=$REPO_ROOT/scripts/run_qemu.sh
OLDPWD=$REPO_ROOT

mkdir -p $BUILDS_DIR
# Download the kernel image if not found
if [[ ! -f "$KERNEL_IMAGE" ]]; then
  echo "No $KERNEL_IMAGE found, prepare to download a new one!"
  release=202310101419
  tmp=$(mktemp -d)
  trap 'rm -rf $tmp' ERR
  curl -SLo $tmp/kernel.tar.gz https://github.com/vietanhduong/kernel-builder/releases/download/${release}/linux-build-${TARGET_KERNEL}.tar.gz &&
    tar -xf $tmp/kernel.tar.gz -C $tmp &&
    cp $tmp/pkg/bzImage $KERNEL_IMAGE
fi

if [[ ! -f "$DISK_IMAGE" ]]; then
  echo "No $DISK_IMAGE found, prepare to download a new one!"
  release=202310121149
  curl -SLo $DISK_IMAGE https://github.com/vietanhduong/qemu-bpf/releases/download/${release}/qemu-${TARGET_KERNEL}.qcow2
fi

# Create a tmp directory that serves as the /test_fs sanbox dir inside qemu.
tmpdir_for_sandbox=$(mktemp -d)
tmp_ssh=$REPO_ROOT/ssh.sh

# shellcheck disable=SC2317
function cleanup {
  retval=$?
  rm -rf "${tmpdir_for_sandbox:?}" || true
  rm -f $tmp_ssh || true

  if [[ -n "${qemu_pid}" ]]; then
    kill "${qemu_pid}" &>/dev/null || true
    wait "${qemu_pid}" || true
    qemu_pid=""
  fi
  exit "${retval}"
}

# We set a trap to make sure the tmp directories are cleaned up.
trap cleanup EXIT
trap cleanup ERR

function strip_pwd_from_path() {
  v=$1
  v=${v//"${OLDPWD}"/""}
  echo "$v"
}

# Replaces the PWD inside qemu with /test_fs.
function path_inside_qemu() {
  v=$1
  test_base="/test_fs"
  v=${v//"${OLDPWD}"/"${test_base}"}
  echo "$v"
}

# Replaces the PWD outside qemu with a path to the sandbox directory.
function path_qemu_sandbox() {
  v=$1
  v=${v//"${OLDPWD}"/"${tmpdir_for_sandbox}"}
  echo "$v"
}

function qemu_is_running() {
  kill -0 "${qemu_pid}" &>/dev/null
}

# We need to write and transform the environment variables so that they have the "correct"
# paths when run inside of qemu.
test_env_file="${tmpdir_for_sandbox}/test_env.sh"
echo "#!/bin/bash" >"${test_env_file}"

# Grab the environment variables and rewrite them for the qemu sandbox.
env -0 | while IFS='=' read -r -d '' n v; do
  # Writing bash funcs as env variables doesn't work. Luckily we don't actually need them.
  if [[ "${n}" == BASH_FUNC* ]]; then
    continue
  fi

  # We don't want to write out the PWD since it will be wrong inside of qemu.
  if [[ "${n}" == PWD ]]; then
    continue
  fi

  if [[ "${n}" == LC_ALL ]]; then
    continue
  fi

  # All paths will basically start with /test_fs.
  v=$(path_inside_qemu "$v")

  printf "export %s=\"%s\"\n" "$n" "$v" >>"${test_env_file}"
done

# Copy the runfiles into the sandbox directory. 9p fs does not like symlinks
# so we resolve them into the copy.
runfiles_path="$(path_qemu_sandbox "${RUNFILES_DIR}")"
mkdir -p "${runfiles_path}"
cp -afL "${RUNFILES_DIR}"/* "${runfiles_path}"
# Copy scripts dir
scripts_path="$(path_qemu_sandbox "${SCRIPTS_DIR}")"
mkdir -p "${scripts_path}"
cp -afL "${SCRIPTS_DIR}"/* "${scripts_path}"

# Create test tmp dir.
if [[ -n "${TEST_TMPDIR}" ]]; then
  mkdir -p "$(path_qemu_sandbox "${TEST_TMPDIR}")"
fi
# Copy the test cmd into the sandbox.
test_base=${PWD//"${OLDPWD}"/\/test_fs}
test_cmd_path="${tmpdir_for_sandbox}/test_cmd.sh"
test_cmd_path_in_qemu="/test_fs/test_cmd.sh"

cat <<EOF >"${test_cmd_path}"
#!/bin/bash -e
cd ${test_base}
if [[ -n "${@:1}" ]]; then
  echo "-------------------------------"
  echo "Command to run test: "
  echo "  ${@:1}"
  echo "-------------------------------"
fi
${@:1}
EOF
chmod +x "${test_cmd_path}"

printf "export test_base=%s\n" "${test_base}" >>"${test_env_file}"
printf "export test_exec_path=%s\n" "${test_cmd_path_in_qemu}" >>"${test_env_file}"

# Setup ssh.
ssh_priv_key="${tmpdir_for_sandbox}/ssh_key"
echo "SSH Private Key location: $ssh_priv_key"
ssh_pub_key_inside_qemu="/test_fs/ssh_key.pub"
ssh-keygen -f "${ssh_priv_key}" -N '' -b 1024 >/dev/null

ssh_env_file="${tmpdir_for_sandbox}/ssh_env.sh"
echo "#!/bin/bash" >"${ssh_env_file}"
echo "export SSH_PUB_KEY=${ssh_pub_key_inside_qemu}" >>"${ssh_env_file}"

run_sshd_file="${tmpdir_for_sandbox}/run_sshd.sh"
cp -afL "${RUN_SSHD_PATH}" "${run_sshd_file}"

monitor_sock="mon.sock"
# Launch qemu.
env - \
  QEMU_TEST_FS_PATH="${tmpdir_for_sandbox}" \
  QEMU_KERNEL_IMAGE="${KERNEL_IMAGE}" \
  QEMU_DISK_BASE_RO="${DISK_IMAGE}" \
  MONITOR_SOCK="${monitor_sock}" \
  "${RUN_QEMU_SCRIPT}" &>"qemu.log" &
qemu_pid="$!"

echo "QEMU logs available at: $(pwd)/qemu.log"
echo "QEMU monitor available at unix socket: $(pwd)/${monitor_sock}"

echo 'Waiting for QEMU to boot'
while qemu_is_running && ! echo "info usernet" | netcat -NU "${monitor_sock}" &>/dev/null; do
  sleep 1
done

host_ssh_port="$(echo "info usernet" | netcat -NU "${monitor_sock}" | grep "HOST_FORWARD" | awk '{print $4}')"
echo "Target SSH Port: ${host_ssh_port}"

ssh_opts=(
  -q
  -i "${ssh_priv_key}"
  -o "StrictHostKeyChecking=no"
  -o "UserKnownHostsFile=/dev/null"
  -p "${host_ssh_port}"
  root@localhost
  -t
)

# Wait for qemu to boot and ssh to be ready
echo 'Waiting for SSH to come online'
while qemu_is_running && ! ssh "${ssh_opts[@]}" 'echo test' &>/dev/null; do
  sleep 1
done

if ! qemu_is_running; then
  echo 'QEMU failed to boot'
  cat -v qemu.log
  echo "Log available in sandbox at: $(pwd)/qemu.log"
  exit 3
fi

cat <<EOF >"${tmp_ssh}"
#!/bin/bash
ssh ${ssh_opts[@]} '/bin/bash -l'
EOF
chmod +x "${tmp_ssh}"
[[ -f "${tmp_ssh}" ]] && echo -e "Tmp SSH: ${tmp_ssh}"

ssh "${ssh_opts[@]}" '/bin/bash -c '"${test_cmd_path_in_qemu}"
