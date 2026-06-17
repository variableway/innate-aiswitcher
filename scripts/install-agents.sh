#!/usr/bin/env bash
set -euo pipefail

# ------------------------------------------------------------------------------
# 鍦?macOS / Linux 涓婁竴閿畨瑁?fnm銆丯ode.js LTS銆丆laude Code銆丆odex CLI銆丱penCode銆?
# ------------------------------------------------------------------------------
# Usage:
#   chmod +x scripts/install-agents.sh
#   ./scripts/install-agents.sh
# ------------------------------------------------------------------------------

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

command_exists() {
    command -v "$1" >/dev/null 2>&1
}

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# ------------------------------------------------------------------------------
# 1. 瀹夎 fnm
# ------------------------------------------------------------------------------
install_fnm() {
    echo ""
    info "[1/6] 瀹夎 fnm ..."
    if command_exists fnm; then
        info "fnm 宸插畨瑁? $(fnm --version)"
        return
    fi

    if command_exists brew; then
        info "閫氳繃 Homebrew 瀹夎 fnm ..."
        brew install fnm
    else
        info "閫氳繃瀹樻柟鑴氭湰瀹夎 fnm ..."
        curl -fsSL https://fnm.vercel.app/install | bash
    fi

    # 鍦ㄥ綋鍓嶄細璇濅腑鍒濆鍖?
    if [ -d "$HOME/.local/share/fnm" ]; then
        export PATH="$HOME/.local/share/fnm:$PATH"
    elif [ -d "$HOME/.fnm" ]; then
        export PATH="$HOME/.fnm:$PATH"
    fi
    eval "$(fnm env --use-on-cd)"

    info "fnm 瀹夎瀹屾垚: $(fnm --version)"
}

# ------------------------------------------------------------------------------
# 2. 鍒濆鍖?fnm 骞跺啓鍏?shell profile
# ------------------------------------------------------------------------------
init_fnm_profile() {
    echo ""
    info "[2/6] 鍒濆鍖?fnm ..."

    eval "$(fnm env --use-on-cd)"

    # 妫€娴?shell profile
    local profile=""
    if [ -n "${ZSH_VERSION:-}" ] || [ -f "$HOME/.zshrc" ]; then
        profile="$HOME/.zshrc"
    elif [ -n "${BASH_VERSION:-}" ] || [ -f "$HOME/.bash_profile" ]; then
        profile="$HOME/.bash_profile"
    else
        profile="$HOME/.profile"
    fi

    local marker="# === innate-aiswitcher: fnm init ==="
    if [ -f "$profile" ] && grep -q "$marker" "$profile" 2>/dev/null; then
        info "Profile 涓凡瀛樺湪 fnm 鍒濆鍖? $profile"
    else
        echo "" >> "$profile"
        echo "$marker" >> "$profile"
        echo 'eval "$(fnm env --use-on-cd)"' >> "$profile"
        echo "# === end ===" >> "$profile"
        info "宸插啓鍏?$profile"
    fi
}

# ------------------------------------------------------------------------------
# 3. 瀹夎 Node.js LTS
# ------------------------------------------------------------------------------
install_node_lts() {
    echo ""
    info "[3/6] 瀹夎 Node.js LTS ..."
    if command_exists node; then
        info "Node.js 宸插瓨鍦? $(node -v)"
    else
        info "瀹夎 Node.js LTS ..."
        fnm install --lts
        fnm use --lts-if-available
    fi
    info "Node.js: $(node -v)"
    info "npm: $(npm -v)"
}

# ------------------------------------------------------------------------------
# 4. 瀹夎 Agents
# ------------------------------------------------------------------------------
install_agent() {
    local name="$1"
    local package="$2"
    local binary="$3"

    echo ""
    info "[瀹夎] $name ($package) ..."
    if npm install -g "$package"; then
        if command_exists "$binary"; then
            info "鎴愬姛: $binary $(${binary} --version 2>&1 | head -n 1)"
            return 0
        else
            warn "宸插畨瑁呬絾鍛戒护 $binary 涓嶅湪 PATH 涓?
            return 1
        fi
    else
        error "$name 瀹夎澶辫触"
        return 1
    fi
}

install_agents() {
    echo ""
    info "[4/6] 瀹夎 AI Agents ..."
    install_agent "Claude Code" "@anthropic-ai/claude-code" "claude" || CLAUDE_OK=false
    install_agent "Codex CLI" "@openai/codex" "codex" || CODEX_OK=false
    install_agent "OpenCode" "opencode" "opencode" || OPENCODE_OK=false
}

# ------------------------------------------------------------------------------
# 5. 涓绘祦绋?
# ------------------------------------------------------------------------------
echo ""
echo "========================================"
echo " AI Agent macOS/Linux 涓€閿畨瑁呰剼鏈?
echo " 鍖呭惈: fnm + Node.js LTS + Claude/Codex/OpenCode"
echo "========================================"

CLAUDE_OK=true
CODEX_OK=true
OPENCODE_OK=true

install_fnm
init_fnm_profile
install_node_lts
install_agents

echo ""
echo "========================================"
echo " 瀹夎缁撴灉"
echo "========================================"

for item in "Claude Code:$CLAUDE_OK" "Codex CLI:$CODEX_OK" "OpenCode:$OPENCODE_OK"; do
    name="${item%%:*}"
    ok="${item##*:}"
    if [ "$ok" = "true" ]; then
        info "$name: OK"
    else
        error "$name: 闇€瑕佹鏌?
    fi
done

echo ""
warn "璇疯繍琛屼互涓嬪懡浠ゅ埛鏂?shell 鐜锛?
echo "  source ~/.zshrc     # 鎴?~/.bash_profile"
echo ""
info "鐒跺悗楠岃瘉锛?
echo "  claude --version"
echo "  codex --version"
echo "  opencode --version"
echo ""
info "涓嬩竴姝ワ細閰嶇疆 innate-aiswitcher"
echo "  ./bin/aisw provider presets"
echo "  ./bin/aisw provider add ..."
echo "  ./bin/aisw profile add ..."
