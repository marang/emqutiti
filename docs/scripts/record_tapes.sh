#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."

# Use an isolated temporary HOME so we never touch the host config.
TMP_HOME="$(mktemp -d -t emqutiti-vhs-home.XXXXXX)"
cleanup() { rm -rf "${TMP_HOME}"; }
trap cleanup EXIT INT TERM
export HOME="${TMP_HOME}"

if ! command -v vhs >/dev/null; then
    echo "vhs is not installed. Run 'make tape' to use the helper container or" >&2
    echo "install it from https://github.com/charmbracelet/vhs" >&2
    exit 1
fi

# Seed a minimal config so demos have a broker profile without
# touching the system keyring inside the container.
seed_config() {
    local cfg_dir="${HOME}/.config/emqutiti"
    mkdir -p "${cfg_dir}"
    # Seed the exact requested brokers into the temporary config.
    # Passwords are left empty to avoid keyring lookups inside containers.
    cat > "${cfg_dir}/config.toml" <<'EOF'
[[profiles]]
  auto_reconnect = true
  ca_cert_path = ""
  clean_start = false
  client_cert_path = ""
  client_id = "emqutiti-user"
  client_key_path = ""
  connect_timeout = 0
  from_env = false
  host = "broker.hivemq.com"
  keep_alive = 0
  last_will_enabled = false
  last_will_payload = ""
  last_will_qos = 0
  last_will_retain = false
  last_will_topic = ""
  maximum_packet_size = 0
  mqtt_version = "5"
  name = "HiveMQ"
  password = ""
  port = 1883
  publish_timeout = 0
  qos = 1
  random_id_suffix = true
  receive_maximum = 0
  reconnect_period = 0
  request_problem_info = false
  request_response_info = false
  schema = "tcp"
  session_expiry_interval = 0
  skip_tls_verify = true
  ssl_tls = true
  subscribe_timeout = 0
  topic_alias_maximum = 0
  unsubscribe_timeout = 0
  username = ""

[[profiles]]
  auto_reconnect = false
  ca_cert_path = ""
  clean_start = false
  client_cert_path = ""
  client_id = ""
  client_key_path = ""
  connect_timeout = 0
  from_env = false
  host = ""
  keep_alive = 0
  last_will_enabled = false
  last_will_payload = ""
  last_will_qos = 0
  last_will_retain = false
  last_will_topic = ""
  maximum_packet_size = 0
  mqtt_version = "3"
  name = "test2"
  password = ""
  port = 0
  publish_timeout = 0
  qos = 0
  random_id_suffix = false
  receive_maximum = 0
  reconnect_period = 0
  request_problem_info = false
  request_response_info = false
  schema = "tcp"
  session_expiry_interval = 0
  skip_tls_verify = false
  ssl_tls = false
  subscribe_timeout = 0
  topic_alias_maximum = 0
  unsubscribe_timeout = 0
  username = ""
EOF
}

seed_config

render_tape() {
    local tape_file=$1
    local gif_file=$2
    echo "Rendering $tape_file to $gif_file..."
    vhs -o "docs/assets/$gif_file" "docs/$tape_file"
    rm -f "docs/${tape_file%.tape}.cast"
}

render_tape client_view.tape client_view.gif
render_tape create_connection.tape create_connection.gif
