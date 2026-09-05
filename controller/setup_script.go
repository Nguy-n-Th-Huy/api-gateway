package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// This file implements GET /api/setup/script (router/api-router.go), which
// turns a checked API key into a one-line install command for a supported AI
// coding CLI. The endpoint returns a plain-text shell script (PowerShell on
// Windows, POSIX shell otherwise) meant to be piped into `iex`/`sh`; the
// frontend only ever builds the URL, it never fetches or parses the script
// itself (see web/src/features/key-check/lib/setup-command.ts).
//
// Every script writes the target CLI's own configuration file rather than
// exporting environment variables, per design.md — Decisions, with Codex as
// the sole documented exception (it reads its credential only from the
// environment).
//
// The two OS variants are genuinely different implementations, not one
// payload in two wrappers:
//   - Windows scripts are pure PowerShell (5.1-compatible: ConvertFrom-Json
//     has no -AsHashtable there, so JSON configs are edited as
//     PSCustomObject graphs). They have no runtime dependency beyond
//     PowerShell itself.
//   - POSIX scripts still run the merge as a small embedded Node.js program
//     (written to a temp file and executed with `node`), because that was
//     the only implementation available when this endpoint shipped. The
//     script now checks for `node` on PATH before doing anything and exits
//     with a clear message if it is missing, per design.md — Decisions.
//
// Both variants share the same file-format editors in spirit (JSON via the
// platform's native parser, a line-oriented TOML section editor for Codex,
// and a `providers:`-block-only YAML editor for Oh My Pi) so the two
// implementations stay behaviorally equivalent.

// ---- supported application registry -------------------------------------
//
// The Windows/unix config file paths and the model-slot keys below come
// verbatim from design.md — Decisions, which is authoritative for this
// change; nothing here invents a key or path that table does not document.
// The `app` id values and the `model_<slot>` query parameter convention are
// fixed by the frontend contract in
// web/src/features/key-check/constants.ts (SETUP_SCRIPT_QUERY_PARAMS,
// APPLICATIONS) — this registry must keep matching it.

type setupScriptOS string

const (
	setupScriptOSWindows setupScriptOS = "windows"
	setupScriptOSUnix    setupScriptOS = "unix"
)

func parseSetupScriptOS(v string) (setupScriptOS, bool) {
	switch setupScriptOS(v) {
	case setupScriptOSWindows, setupScriptOSUnix:
		return setupScriptOS(v), true
	default:
		return "", false
	}
}

type setupScriptApp string

const (
	setupScriptAppClaudeCode setupScriptApp = "claude-code"
	setupScriptAppCodex      setupScriptApp = "codex"
	setupScriptAppOpenCode   setupScriptApp = "opencode"
	setupScriptAppPi         setupScriptApp = "pi"
	setupScriptAppOhMyPi     setupScriptApp = "oh-my-pi"
)

// setupScriptAppInfo documents one supported application. The frontend
// renders its own install step independently (see
// web/src/features/key-check/lib/applications.ts) — this registry only
// needs what the script itself must know.
type setupScriptAppInfo struct {
	DisplayName string
	// Slots lists this application's model-role slot keys, in display order.
	// The setup-script query parameter for a slot is "model_<slot>". Empty
	// for an application with no documented model-role configuration (Pi,
	// Oh My Pi); their script instead registers every model the key's group
	// enables and leaves role selection to the CLI itself.
	Slots []string
	// ConfigPath is the documented, human-readable path for messages and
	// tests. The generated script itself resolves the real path at runtime
	// from the user's home directory, so it is correct for any account name.
	ConfigPath map[setupScriptOS]string
}

var setupScriptAppOrder = []setupScriptApp{
	setupScriptAppClaudeCode,
	setupScriptAppCodex,
	setupScriptAppOpenCode,
	setupScriptAppPi,
	setupScriptAppOhMyPi,
}

var setupScriptApps = map[setupScriptApp]setupScriptAppInfo{
	setupScriptAppClaudeCode: {
		DisplayName: "Claude Code",
		Slots:       []string{"opus", "sonnet", "haiku", "subagent"},
		ConfigPath: map[setupScriptOS]string{
			setupScriptOSUnix:    "~/.claude/settings.json",
			setupScriptOSWindows: `%USERPROFILE%\.claude\settings.json`,
		},
	},
	setupScriptAppCodex: {
		DisplayName: "Codex",
		Slots:       []string{"small", "medium", "large"},
		ConfigPath: map[setupScriptOS]string{
			setupScriptOSUnix:    "~/.codex/config.toml",
			setupScriptOSWindows: `%USERPROFILE%\.codex\config.toml`,
		},
	},
	setupScriptAppOpenCode: {
		DisplayName: "OpenCode",
		Slots:       []string{"model", "small_model"},
		ConfigPath: map[setupScriptOS]string{
			setupScriptOSUnix:    "~/.config/opencode/opencode.json",
			setupScriptOSWindows: `%USERPROFILE%\.config\opencode\opencode.json`,
		},
	},
	setupScriptAppPi: {
		DisplayName: "Pi",
		Slots:       nil,
		ConfigPath: map[setupScriptOS]string{
			setupScriptOSUnix:    "~/.pi/agent/models.json",
			setupScriptOSWindows: `%USERPROFILE%\.pi\agent\models.json`,
		},
	},
	setupScriptAppOhMyPi: {
		DisplayName: "Oh My Pi",
		Slots:       nil,
		ConfigPath: map[setupScriptOS]string{
			setupScriptOSUnix:    "~/.omp/agent/models.yml",
			setupScriptOSWindows: `%USERPROFILE%\.omp\agent\models.yml`,
		},
	},
}

// newapiProviderID is the fixed provider/id key every generated script
// registers this gateway under. It is internal to the generated config files
// (never exposed as API surface), so a stable, readable identifier is enough.
const newapiProviderID = "newapi"

// requestOrigin derives "scheme://host" from the incoming request. It is the
// fallback base URL when system_setting.ServerAddress has never been
// configured by an operator (see design.md — Decisions), mirroring the
// scheme/host resolution the frontend already applies via
// window.location.origin and the one isSelfTaskMediaURL applies for
// reverse-proxy deployments (controller/video_proxy.go).
func requestOrigin(c *gin.Context) string {
	scheme := strings.TrimSpace(strings.Split(c.Request.Header.Get("X-Forwarded-Proto"), ",")[0])
	if scheme == "" {
		scheme = "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
	}
	host := strings.TrimSpace(strings.Split(c.Request.Header.Get("X-Forwarded-Host"), ",")[0])
	if host == "" {
		host = c.Request.Host
	}
	if host == "" {
		return ""
	}
	return scheme + "://" + host
}

// ---- string-literal encoding helpers --------------------------------------

// jsString renders a Go string as a JS double-quoted string literal. A JSON
// string is always a valid JS string literal for the same input, so
// common.Marshal (the project's required JSON entry point) does the escaping.
func jsString(s string) string {
	b, _ := common.Marshal(s)
	return string(b)
}

// tomlString renders a Go string as a TOML basic (double-quoted) string.
// TOML's basic-string escaping matches JSON's for every character this
// feature ever emits (ids, model names, URLs), so JSON-encoding is safe here.
func tomlString(s string) string {
	b, _ := common.Marshal(s)
	return string(b)
}

// yamlString renders a Go string as a YAML double-quoted flow scalar. Same
// reasoning as tomlString: JSON escaping is a safe subset of YAML's.
func yamlString(s string) string {
	b, _ := common.Marshal(s)
	return string(b)
}

// jsHomedirPath renders a Node.js expression that joins the user's home
// directory with the given path segments via path.join, so the generated
// script resolves the real path on whichever machine runs it.
func jsHomedirPath(segments ...string) string {
	parts := make([]string, 0, len(segments)+1)
	parts = append(parts, "os.homedir()")
	for _, s := range segments {
		parts = append(parts, jsString(s))
	}
	return "path.join(" + strings.Join(parts, ", ") + ")"
}

// psHomedirPath renders a PowerShell double-quoted string that interpolates
// $env:USERPROFILE with the given path segments. Segments are always this
// file's own literal path components (never user input), so no escaping is
// needed beyond the backslash join.
func psHomedirPath(segments ...string) string {
	return `"$env:USERPROFILE\` + strings.Join(segments, `\`) + `"`
}

// posixSingleQuote renders s as a single-quoted POSIX shell word, safe
// regardless of its contents.
func posixSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// posixDoubleQuoted renders s as a double-quoted POSIX shell string body
// (including the surrounding quotes), escaping the characters double quotes
// still expand: backslash, double quote, `$` and backtick.
func posixDoubleQuoted(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "$", "\\$", "`", "\\`")
	return `"` + replacer.Replace(s) + `"`
}

// psSingleQuote renders s as a single-quoted PowerShell string literal, safe
// regardless of its contents (a literal single quote doubles).
func psSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// psStringArrayLiteral renders a Go []string as a PowerShell array literal
// of single-quoted string elements, safe regardless of their contents. An
// empty slice renders as `@()` (an empty PowerShell array), never a bare
// value that PowerShell could unwrap to $null.
func psStringArrayLiteral(values []string) string {
	if len(values) == 0 {
		return "@()"
	}
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = psSingleQuote(v)
	}
	return "@(" + strings.Join(parts, ", ") + ")"
}

// ---- JSON-config apps (Claude Code, OpenCode, Pi) — Node.js / POSIX --------
//
// Node has JSON.parse/JSON.stringify built in, so these scripts merge the
// owned keys through a real parsed object rather than any text surgery: it
// is the most direct way to satisfy "update only the keys this change owns,
// leave every other key intact" for a JSON file.

const jsonConfigPreamble = `const fs = require("fs");
const os = require("os");
const path = require("path");

const configPath = %s;
fs.mkdirSync(path.dirname(configPath), { recursive: true });

let config = {};
if (fs.existsSync(configPath)) {
  const raw = fs.readFileSync(configPath, "utf8");
  if (raw.trim() !== "") {
    try {
      config = JSON.parse(raw);
    } catch (e) {
      console.error("Could not parse existing configuration file: " + configPath);
      process.exit(1);
    }
  }
}
if (typeof config !== "object" || config === null || Array.isArray(config)) {
  console.error("Could not parse existing configuration file: " + configPath);
  process.exit(1);
}
`

func jsonConfigScript(configPathExpr string, mergeCode string, successLabel string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(jsonConfigPreamble, configPathExpr))
	b.WriteString(mergeCode)
	b.WriteString(fmt.Sprintf(`
fs.writeFileSync(configPath, JSON.stringify(config, null, 2) + "\n", "utf8");
console.log(%s + " configured: " + configPath);
`, jsString(successLabel)))
	return b.String()
}

func buildClaudeCodeScript(baseURL, key string, models map[string]string) string {
	mergeCode := fmt.Sprintf(`if (typeof config.env !== "object" || config.env === null || Array.isArray(config.env)) {
  config.env = {};
}
config.env.ANTHROPIC_BASE_URL = %s;
config.env.ANTHROPIC_AUTH_TOKEN = %s;
config.env.ANTHROPIC_DEFAULT_OPUS_MODEL = %s;
config.env.ANTHROPIC_DEFAULT_SONNET_MODEL = %s;
config.env.ANTHROPIC_DEFAULT_HAIKU_MODEL = %s;
config.env.CLAUDE_CODE_SUBAGENT_MODEL = %s;
`,
		jsString(baseURL), jsString(key),
		jsString(models["opus"]), jsString(models["sonnet"]), jsString(models["haiku"]), jsString(models["subagent"]))
	return jsonConfigScript(jsHomedirPath(".claude", "settings.json"), mergeCode, "Claude Code")
}

func buildOpenCodeScript(baseURL, key string, models map[string]string, availableModels []string) string {
	modelsJSON, _ := common.Marshal(availableModels)
	modelRef := jsString("newapi/" + models["model"])
	smallModelRef := jsString("newapi/" + models["small_model"])
	mergeCode := fmt.Sprintf(`if (typeof config.provider !== "object" || config.provider === null || Array.isArray(config.provider)) {
  config.provider = {};
}
const newapiModels = {};
for (const m of %s) {
  newapiModels[m] = {};
}
config.provider.newapi = {
  npm: "@ai-sdk/openai-compatible",
  name: "New API",
  options: { baseURL: %s, apiKey: %s },
  models: newapiModels,
};
config.model = %s;
config.small_model = %s;
`, string(modelsJSON), jsString(baseURL), jsString(key), modelRef, smallModelRef)
	return jsonConfigScript(jsHomedirPath(".config", "opencode", "opencode.json"), mergeCode, "OpenCode")
}

func buildPiScript(baseURL, key string, availableModels []string) string {
	modelsJSON, _ := common.Marshal(availableModels)
	mergeCode := fmt.Sprintf(`if (typeof config.providers !== "object" || config.providers === null || Array.isArray(config.providers)) {
  config.providers = {};
}
config.providers.newapi = {
  baseUrl: %s,
  api: "openai-completions",
  apiKey: %s,
  models: (%s).map((m) => ({ id: m })),
};
`, jsString(baseURL), jsString(key), string(modelsJSON))
	return jsonConfigScript(jsHomedirPath(".pi", "agent", "models.json"), mergeCode, "Pi")
}

// ---- JSON-config apps (Claude Code, OpenCode, Pi) — PowerShell / Windows --
//
// PowerShell 5.1's ConvertFrom-Json returns PSCustomObject graphs (no
// -AsHashtable there), and dot-assignment fails on a property that does not
// already exist, so every mutation goes through Set-ConfigProperty
// (Add-Member -Force, which both creates and overwrites) rather than plain
// dot-assignment.

const psJSONHelpers = `function New-ConfigObject {
    New-Object -TypeName PSObject
}

function Set-ConfigProperty {
    param($Target, [string]$Name, $Value)
    $Target | Add-Member -NotePropertyName $Name -NotePropertyValue $Value -Force
}

function Get-OrCreateObjectProperty {
    param($Target, [string]$Name)
    $existing = $Target.PSObject.Properties[$Name]
    if ($existing -and ($existing.Value -is [System.Management.Automation.PSCustomObject])) {
        return $existing.Value
    }
    $created = New-ConfigObject
    Set-ConfigProperty -Target $Target -Name $Name -Value $created
    return $created
}

function Read-JsonConfig {
    param([string]$ConfigPath)
    $dir = Split-Path -Path $ConfigPath -Parent
    [System.IO.Directory]::CreateDirectory($dir) | Out-Null
    $config = $null
    if (Test-Path -LiteralPath $ConfigPath) {
        $raw = [System.IO.File]::ReadAllText($ConfigPath)
        if ($raw.Trim() -ne '') {
            try {
                $config = $raw | ConvertFrom-Json -ErrorAction Stop
            } catch {
                [Console]::Error.WriteLine("Could not parse existing configuration file: " + $ConfigPath)
                exit 1
            }
        }
    }
    if ($null -eq $config) {
        return New-ConfigObject
    }
    if (-not ($config -is [System.Management.Automation.PSCustomObject])) {
        [Console]::Error.WriteLine("Could not parse existing configuration file: " + $ConfigPath)
        exit 1
    }
    return $config
}

function Write-JsonConfig {
    param([string]$ConfigPath, $Config, [string]$SuccessLabel)
    $json = $Config | ConvertTo-Json -Depth 20
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText($ConfigPath, $json + [string][char]10, $utf8NoBom)
    Write-Host ($SuccessLabel + " configured: " + $ConfigPath)
}
`

func psJsonConfigScript(configPathExpr string, mergeCode string, successLabel string) string {
	var b strings.Builder
	b.WriteString(psJSONHelpers)
	b.WriteString("\n$configPath = " + configPathExpr + "\n$config = Read-JsonConfig -ConfigPath $configPath\n")
	b.WriteString(mergeCode)
	b.WriteString("\nWrite-JsonConfig -ConfigPath $configPath -Config $config -SuccessLabel " + psSingleQuote(successLabel) + "\n")
	return b.String()
}

func buildClaudeCodeScriptPS(baseURL, key string, models map[string]string) string {
	mergeCode := "$envObj = Get-OrCreateObjectProperty -Target $config -Name 'env'\n" +
		"Set-ConfigProperty -Target $envObj -Name 'ANTHROPIC_BASE_URL' -Value " + psSingleQuote(baseURL) + "\n" +
		"Set-ConfigProperty -Target $envObj -Name 'ANTHROPIC_AUTH_TOKEN' -Value " + psSingleQuote(key) + "\n" +
		"Set-ConfigProperty -Target $envObj -Name 'ANTHROPIC_DEFAULT_OPUS_MODEL' -Value " + psSingleQuote(models["opus"]) + "\n" +
		"Set-ConfigProperty -Target $envObj -Name 'ANTHROPIC_DEFAULT_SONNET_MODEL' -Value " + psSingleQuote(models["sonnet"]) + "\n" +
		"Set-ConfigProperty -Target $envObj -Name 'ANTHROPIC_DEFAULT_HAIKU_MODEL' -Value " + psSingleQuote(models["haiku"]) + "\n" +
		"Set-ConfigProperty -Target $envObj -Name 'CLAUDE_CODE_SUBAGENT_MODEL' -Value " + psSingleQuote(models["subagent"]) + "\n"
	return psJsonConfigScript(psHomedirPath(".claude", "settings.json"), mergeCode, "Claude Code")
}

func buildOpenCodeScriptPS(baseURL, key string, models map[string]string, availableModels []string) string {
	modelRef := "newapi/" + models["model"]
	smallModelRef := "newapi/" + models["small_model"]
	mergeCode := "$providerContainer = Get-OrCreateObjectProperty -Target $config -Name 'provider'\n" +
		"$newapiModels = New-ConfigObject\n" +
		"foreach ($m in " + psStringArrayLiteral(availableModels) + ") {\n" +
		"    Set-ConfigProperty -Target $newapiModels -Name $m -Value (New-ConfigObject)\n" +
		"}\n" +
		"$optionsObj = New-ConfigObject\n" +
		"Set-ConfigProperty -Target $optionsObj -Name 'baseURL' -Value " + psSingleQuote(baseURL) + "\n" +
		"Set-ConfigProperty -Target $optionsObj -Name 'apiKey' -Value " + psSingleQuote(key) + "\n" +
		"$newapiProvider = New-ConfigObject\n" +
		"Set-ConfigProperty -Target $newapiProvider -Name 'npm' -Value '@ai-sdk/openai-compatible'\n" +
		"Set-ConfigProperty -Target $newapiProvider -Name 'name' -Value 'New API'\n" +
		"Set-ConfigProperty -Target $newapiProvider -Name 'options' -Value $optionsObj\n" +
		"Set-ConfigProperty -Target $newapiProvider -Name 'models' -Value $newapiModels\n" +
		"Set-ConfigProperty -Target $providerContainer -Name 'newapi' -Value $newapiProvider\n" +
		"Set-ConfigProperty -Target $config -Name 'model' -Value " + psSingleQuote(modelRef) + "\n" +
		"Set-ConfigProperty -Target $config -Name 'small_model' -Value " + psSingleQuote(smallModelRef) + "\n"
	return psJsonConfigScript(psHomedirPath(".config", "opencode", "opencode.json"), mergeCode, "OpenCode")
}

func buildPiScriptPS(baseURL, key string, availableModels []string) string {
	mergeCode := "$providersObj = Get-OrCreateObjectProperty -Target $config -Name 'providers'\n" +
		"$modelObjs = @()\n" +
		"foreach ($m in " + psStringArrayLiteral(availableModels) + ") {\n" +
		"    $modelObj = New-ConfigObject\n" +
		"    Set-ConfigProperty -Target $modelObj -Name 'id' -Value $m\n" +
		"    $modelObjs = @($modelObjs) + @($modelObj)\n" +
		"}\n" +
		"$newapiProvider = New-ConfigObject\n" +
		"Set-ConfigProperty -Target $newapiProvider -Name 'baseUrl' -Value " + psSingleQuote(baseURL) + "\n" +
		"Set-ConfigProperty -Target $newapiProvider -Name 'api' -Value 'openai-completions'\n" +
		"Set-ConfigProperty -Target $newapiProvider -Name 'apiKey' -Value " + psSingleQuote(key) + "\n" +
		"Set-ConfigProperty -Target $newapiProvider -Name 'models' -Value @($modelObjs)\n" +
		"Set-ConfigProperty -Target $providersObj -Name 'newapi' -Value $newapiProvider\n"
	return psJsonConfigScript(psHomedirPath(".pi", "agent", "models.json"), mergeCode, "Pi")
}

// ---- Codex (TOML) — Node.js / POSIX ----------------------------------------
//
// Node has no built-in TOML support. Rather than hand-roll a full TOML
// parser, tomlEditorJS is a line-oriented section editor: it recognizes only
// [table]/[[array table]] headers and top-level "key = value" assignments
// well enough to replace the sections and top-level keys this change owns,
// leaving every other section, key and comment byte-for-byte untouched.
//
// basicSanityCheck accepts tab characters (legal TOML whitespace) and
// multi-line arrays: netBracketDelta tracks the net "[" / "]" depth of a
// line while ignoring bracket characters that appear inside a quoted
// string, so an assignment like `args = [\n  "-y",\n  "everything"\n]`
// (real Codex `[mcp_servers.*]` entries look exactly like this) is treated
// as one continuing logical statement instead of being rejected line by
// line. editToml itself needs no special handling for this: none of the
// keys this change owns are ever multi-line, so a continuation line simply
// never matches an owned top-level key and is always carried through
// verbatim. A basic sanity pass still rejects lines that are neither a
// header nor an assignment and an unterminated array/triple-quoted string,
// so a file that is not safely editable stops the script instead of being
// silently corrupted.
const tomlEditorJS = `function netBracketDelta(line) {
  let depth = 0;
  let quote = null;
  for (let i = 0; i < line.length; i++) {
    const ch = line[i];
    if (quote) {
      if (quote === '"' && ch === "\\") { i++; continue; }
      if (ch === quote) { quote = null; }
      continue;
    }
    if (ch === '"' || ch === "'") { quote = ch; continue; }
    if (ch === "#") break;
    if (ch === "[") depth++;
    else if (ch === "]") depth--;
  }
  return depth;
}

function basicSanityCheck(text) {
  const lines = text.split(/\r\n|\n/);
  let openTripleQuote = false;
  let arrayDepth = 0;
  for (const line of lines) {
    const stripped = line.trim();
    if (openTripleQuote) {
      if (stripped.indexOf('"""') !== -1 || stripped.indexOf("'''") !== -1) {
        openTripleQuote = false;
      }
      continue;
    }
    if (arrayDepth > 0) {
      arrayDepth += netBracketDelta(line);
      if (arrayDepth < 0) arrayDepth = 0;
      continue;
    }
    if (stripped === "" || stripped.startsWith("#")) continue;
    const isHeader = /^\[\[?[^\[\]]+\]\]?$/.test(stripped);
    const isAssignment = /^[^=\s][^=]*=.+$/.test(stripped);
    if (!isHeader && !isAssignment) {
      return "line does not look like a TOML header or assignment: " + JSON.stringify(line);
    }
    if (isAssignment) {
      const delta = netBracketDelta(stripped);
      if (delta > 0) {
        arrayDepth = delta;
      }
    }
    const tripleCount = (stripped.match(/"""/g) || []).length + (stripped.match(/'''/g) || []).length;
    if (tripleCount % 2 === 1) {
      openTripleQuote = true;
    }
  }
  if (openTripleQuote) {
    return "file has an unterminated triple-quoted string";
  }
  if (arrayDepth > 0) {
    return "file has an unterminated array";
  }
  return null;
}

function editToml(existingText, ownedSections, ownedTopLevelKeys, ownedSectionBlocks, ownedTopLevelLines) {
  let text = existingText || "";
  if (text.trim() !== "") {
    const problem = basicSanityCheck(text);
    if (problem) {
      throw new Error(problem);
    }
  }

  const lines = text.length ? text.split(/\r\n|\n/) : [];
  const keptLines = [];
  let currentSection = null;
  let skippingOwnedSection = false;

  for (const line of lines) {
    const stripped = line.trim();
    const headerMatch = stripped.match(/^\[\[?([^\[\]]+)\]\]?$/);
    if (headerMatch) {
      currentSection = headerMatch[1].trim();
      skippingOwnedSection = ownedSections.indexOf(currentSection) !== -1;
      if (skippingOwnedSection) continue;
      keptLines.push(line);
      continue;
    }
    if (skippingOwnedSection) {
      continue;
    }
    if (currentSection === null) {
      const kvMatch = stripped.match(/^([^=\s][^=]*?)\s*=/);
      if (kvMatch && ownedTopLevelKeys.indexOf(kvMatch[1].trim()) !== -1) {
        continue;
      }
    }
    keptLines.push(line);
  }

  while (keptLines.length && keptLines[keptLines.length - 1].trim() === "") {
    keptLines.pop();
  }

  const out = [];
  out.push(...ownedTopLevelLines);
  if (keptLines.length) {
    out.push("");
    out.push(...keptLines);
  }
  out.push("");
  out.push(...ownedSectionBlocks);
  return out.join("\n") + "\n";
}
`

func buildCodexScript(baseURL, key string, models map[string]string) string {
	baseURLWithV1 := strings.TrimRight(baseURL, "/") + "/v1"
	envKey := "NEWAPI_API_KEY"

	ownedSections := []string{
		"model_providers." + newapiProviderID,
		"profiles.small",
		"profiles.medium",
		"profiles.large",
	}
	ownedTopLevelKeys := []string{"model", "model_provider"}
	ownedTopLevelLines := []string{
		fmt.Sprintf("model = %s", tomlString(models["medium"])),
		fmt.Sprintf("model_provider = %s", tomlString(newapiProviderID)),
	}
	ownedSectionBlocks := []string{
		fmt.Sprintf("[model_providers.%s]", newapiProviderID),
		fmt.Sprintf("name = %s", tomlString("New API")),
		fmt.Sprintf("base_url = %s", tomlString(baseURLWithV1)),
		`wire_api = "responses"`,
		fmt.Sprintf("env_key = %s", tomlString(envKey)),
		"",
		"[profiles.small]",
		fmt.Sprintf("model_provider = %s", tomlString(newapiProviderID)),
		fmt.Sprintf("model = %s", tomlString(models["small"])),
		"",
		"[profiles.medium]",
		fmt.Sprintf("model_provider = %s", tomlString(newapiProviderID)),
		fmt.Sprintf("model = %s", tomlString(models["medium"])),
		"",
		"[profiles.large]",
		fmt.Sprintf("model_provider = %s", tomlString(newapiProviderID)),
		fmt.Sprintf("model = %s", tomlString(models["large"])),
	}

	ownedSectionsJSON, _ := common.Marshal(ownedSections)
	ownedTopLevelKeysJSON, _ := common.Marshal(ownedTopLevelKeys)
	ownedSectionBlocksJSON, _ := common.Marshal(ownedSectionBlocks)
	ownedTopLevelLinesJSON, _ := common.Marshal(ownedTopLevelLines)

	var b strings.Builder
	b.WriteString("const fs = require(\"fs\");\nconst os = require(\"os\");\nconst path = require(\"path\");\n\n")
	b.WriteString(tomlEditorJS)
	b.WriteString(fmt.Sprintf(`
const configPath = %s;
fs.mkdirSync(path.dirname(configPath), { recursive: true });
let existing = "";
if (fs.existsSync(configPath)) {
  existing = fs.readFileSync(configPath, "utf8");
}
let output;
try {
  output = editToml(existing, %s, %s, %s, %s);
} catch (e) {
  console.error("Could not parse existing configuration file: " + configPath + " (" + e.message + ")");
  process.exit(1);
}
fs.writeFileSync(configPath, output, "utf8");
console.log("Codex configured: " + configPath);
`, jsHomedirPath(".codex", "config.toml"),
		string(ownedSectionsJSON), string(ownedTopLevelKeysJSON), string(ownedSectionBlocksJSON), string(ownedTopLevelLinesJSON)))
	return b.String()
}

// ---- Codex (TOML) — PowerShell / Windows -----------------------------------
//
// A direct port of tomlEditorJS/basicSanityCheck above, so the two
// implementations accept and reject exactly the same files. PowerShell
// backtick escapes (`n, `t, `r) are avoided throughout — they would collide
// with Go's backtick-delimited raw string literal used to embed this text —
// in favor of [char] codes and .NET regex escape sequences, which PowerShell
// passes through to the regex engine unmodified.
const psTomlEditor = `function Get-NetBracketDelta {
    param([string]$Line)
    $depth = 0
    $quote = $null
    $chars = $Line.ToCharArray()
    $i = 0
    while ($i -lt $chars.Length) {
        $ch = $chars[$i]
        if ($quote) {
            if ($quote -eq '"' -and $ch -eq '\') {
                $i += 2
                continue
            }
            if ($ch -eq $quote) { $quote = $null }
            $i++
            continue
        }
        if ($ch -eq '"' -or $ch -eq "'") { $quote = $ch; $i++; continue }
        if ($ch -eq '#') { break }
        if ($ch -eq '[') { $depth++ }
        elseif ($ch -eq ']') { $depth-- }
        $i++
    }
    return $depth
}

function Test-TomlSanity {
    param([string]$Text)
    $lines = [regex]::Split($Text, '\r\n|\n')
    $openTripleQuote = $false
    $arrayDepth = 0
    foreach ($line in $lines) {
        $stripped = $line.Trim()
        if ($openTripleQuote) {
            if ($stripped.Contains('"""') -or $stripped.Contains("'''")) { $openTripleQuote = $false }
            continue
        }
        if ($arrayDepth -gt 0) {
            $arrayDepth += Get-NetBracketDelta $line
            if ($arrayDepth -lt 0) { $arrayDepth = 0 }
            continue
        }
        if ($stripped -eq '' -or $stripped.StartsWith('#')) { continue }
        $isHeader = $stripped -match '^\[\[?[^\[\]]+\]\]?$'
        $isAssignment = $stripped -match '^[^=\s][^=]*=.+$'
        if (-not $isHeader -and -not $isAssignment) {
            return "line does not look like a TOML header or assignment: " + $line
        }
        if ($isAssignment) {
            $delta = Get-NetBracketDelta $stripped
            if ($delta -gt 0) { $arrayDepth = $delta }
        }
        $tripleCount = ([regex]::Matches($stripped, '"""')).Count + ([regex]::Matches($stripped, "'''")).Count
        if ($tripleCount % 2 -eq 1) { $openTripleQuote = $true }
    }
    if ($openTripleQuote) { return "file has an unterminated triple-quoted string" }
    if ($arrayDepth -gt 0) { return "file has an unterminated array" }
    return $null
}

function Edit-Toml {
    param(
        [string]$ExistingText,
        [string[]]$OwnedSections,
        [string[]]$OwnedTopLevelKeys,
        [string[]]$OwnedSectionBlocks,
        [string[]]$OwnedTopLevelLines
    )
    $text = if ($ExistingText) { $ExistingText } else { '' }
    if ($text.Trim() -ne '') {
        $problem = Test-TomlSanity $text
        if ($problem) { throw $problem }
    }

    $lines = if ($text.Length -gt 0) { [regex]::Split($text, '\r\n|\n') } else { @() }
    $keptLines = New-Object System.Collections.Generic.List[string]
    $currentSection = $null
    $skippingOwnedSection = $false

    foreach ($line in $lines) {
        $stripped = $line.Trim()
        $headerMatch = [regex]::Match($stripped, '^\[\[?([^\[\]]+)\]\]?$')
        if ($headerMatch.Success) {
            $currentSection = $headerMatch.Groups[1].Value.Trim()
            $skippingOwnedSection = $OwnedSections -contains $currentSection
            if ($skippingOwnedSection) { continue }
            $keptLines.Add($line)
            continue
        }
        if ($skippingOwnedSection) { continue }
        if ($null -eq $currentSection) {
            $kvMatch = [regex]::Match($stripped, '^([^=\s][^=]*?)\s*=')
            if ($kvMatch.Success -and ($OwnedTopLevelKeys -contains $kvMatch.Groups[1].Value.Trim())) {
                continue
            }
        }
        $keptLines.Add($line)
    }

    while ($keptLines.Count -gt 0 -and $keptLines[$keptLines.Count - 1].Trim() -eq '') {
        $keptLines.RemoveAt($keptLines.Count - 1)
    }

    $out = New-Object System.Collections.Generic.List[string]
    foreach ($l in $OwnedTopLevelLines) { $out.Add($l) }
    if ($keptLines.Count -gt 0) {
        $out.Add('')
        foreach ($l in $keptLines) { $out.Add($l) }
    }
    $out.Add('')
    foreach ($l in $OwnedSectionBlocks) { $out.Add($l) }

    return ($out -join [string][char]10) + [string][char]10
}
`

func buildCodexScriptPS(baseURL, key string, models map[string]string) string {
	baseURLWithV1 := strings.TrimRight(baseURL, "/") + "/v1"
	envKey := "NEWAPI_API_KEY"

	ownedSections := []string{
		"model_providers." + newapiProviderID,
		"profiles.small",
		"profiles.medium",
		"profiles.large",
	}
	ownedTopLevelKeys := []string{"model", "model_provider"}
	ownedTopLevelLines := []string{
		fmt.Sprintf("model = %s", tomlString(models["medium"])),
		fmt.Sprintf("model_provider = %s", tomlString(newapiProviderID)),
	}
	ownedSectionBlocks := []string{
		fmt.Sprintf("[model_providers.%s]", newapiProviderID),
		fmt.Sprintf("name = %s", tomlString("New API")),
		fmt.Sprintf("base_url = %s", tomlString(baseURLWithV1)),
		`wire_api = "responses"`,
		fmt.Sprintf("env_key = %s", tomlString(envKey)),
		"",
		"[profiles.small]",
		fmt.Sprintf("model_provider = %s", tomlString(newapiProviderID)),
		fmt.Sprintf("model = %s", tomlString(models["small"])),
		"",
		"[profiles.medium]",
		fmt.Sprintf("model_provider = %s", tomlString(newapiProviderID)),
		fmt.Sprintf("model = %s", tomlString(models["medium"])),
		"",
		"[profiles.large]",
		fmt.Sprintf("model_provider = %s", tomlString(newapiProviderID)),
		fmt.Sprintf("model = %s", tomlString(models["large"])),
	}

	var b strings.Builder
	b.WriteString(psTomlEditor)
	b.WriteString("\n$configPath = " + psHomedirPath(".codex", "config.toml") + "\n")
	b.WriteString("$dir = Split-Path -Path $configPath -Parent\n")
	b.WriteString("[System.IO.Directory]::CreateDirectory($dir) | Out-Null\n")
	b.WriteString("$existing = ''\n")
	b.WriteString("if (Test-Path -LiteralPath $configPath) {\n    $existing = [System.IO.File]::ReadAllText($configPath)\n}\n")
	b.WriteString("try {\n")
	b.WriteString("    $output = Edit-Toml -ExistingText $existing -OwnedSections " + psStringArrayLiteral(ownedSections) +
		" -OwnedTopLevelKeys " + psStringArrayLiteral(ownedTopLevelKeys) +
		" -OwnedSectionBlocks " + psStringArrayLiteral(ownedSectionBlocks) +
		" -OwnedTopLevelLines " + psStringArrayLiteral(ownedTopLevelLines) + "\n")
	b.WriteString("} catch {\n")
	b.WriteString("    [Console]::Error.WriteLine(\"Could not parse existing configuration file: \" + $configPath + \" (\" + $_.Exception.Message + \")\")\n")
	b.WriteString("    exit 1\n")
	b.WriteString("}\n")
	b.WriteString("$utf8NoBom = New-Object System.Text.UTF8Encoding($false)\n")
	b.WriteString("[System.IO.File]::WriteAllText($configPath, $output, $utf8NoBom)\n")
	b.WriteString("Write-Host (\"Codex configured: \" + $configPath)\n")
	return b.String()
}

// ---- Oh My Pi (YAML) — Node.js / POSIX -------------------------------------
//
// design.md restricts this script to the documented `providers` root key
// (an unknown root key fails that file's schema validation), so
// yamlProvidersEditorJS only ever touches that one top-level block and,
// within it, only this gateway's own provider sub-block — every other
// top-level key and every sibling provider entry is preserved verbatim.
const yamlProvidersEditorJS = `function basicSanityCheck(text) {
  if (text.indexOf("\t") !== -1) {
    return "file contains tab characters, which is not valid YAML indentation";
  }
  return null;
}

function topLevelKeyLine(line) {
  return /^[A-Za-z0-9_.-]+:\s*(#.*)?$/.test(line) || /^[A-Za-z0-9_.-]+:\s+\S.*$/.test(line);
}

function providerEntryLine(line) {
  const m = line.match(/^  ([A-Za-z0-9_.-]+):\s*(#.*)?$/);
  return m ? m[1] : null;
}

function editProvidersYaml(existingText, providerId, providerBlockLines) {
  let text = existingText || "";
  if (text.trim() !== "") {
    const problem = basicSanityCheck(text);
    if (problem) throw new Error(problem);
  }

  const lines = text.length ? text.split(/\r\n|\n/) : [];

  let providersStart = -1;
  let providersEnd = lines.length;
  for (let i = 0; i < lines.length; i++) {
    if (lines[i] === "providers:") {
      providersStart = i;
      for (let j = i + 1; j < lines.length; j++) {
        if (topLevelKeyLine(lines[j])) {
          providersEnd = j;
          break;
        }
      }
      break;
    }
  }

  const before = providersStart === -1 ? lines.slice() : lines.slice(0, providersStart);
  const after = providersStart === -1 ? [] : lines.slice(providersEnd);
  const providersBody = providersStart === -1 ? [] : lines.slice(providersStart + 1, providersEnd);

  const entries = [];
  let i = 0;
  while (i < providersBody.length) {
    const id = providerEntryLine(providersBody[i]);
    if (id) {
      let j = i + 1;
      while (j < providersBody.length && providerEntryLine(providersBody[j]) === null && (providersBody[j].startsWith("   ") || providersBody[j].trim() === "")) {
        j++;
      }
      entries.push({ id, start: i, end: j });
      i = j;
    } else {
      i++;
    }
  }

  let replaced = false;
  const newBody = [];
  let cursor = 0;
  for (const entry of entries) {
    newBody.push(...providersBody.slice(cursor, entry.start));
    if (entry.id === providerId) {
      newBody.push(...providerBlockLines);
      replaced = true;
    } else {
      newBody.push(...providersBody.slice(entry.start, entry.end));
    }
    cursor = entry.end;
  }
  newBody.push(...providersBody.slice(cursor));
  if (!replaced) {
    newBody.push(...providerBlockLines);
  }

  const out = [];
  out.push(...before);
  if (before.length && before[before.length - 1].trim() !== "") out.push("");
  out.push("providers:");
  out.push(...newBody);
  if (after.length) {
    out.push(...after);
  }

  while (out.length > 1 && out[out.length - 1].trim() === "" && out[out.length - 2].trim() === "") {
    out.pop();
  }
  return out.join("\n").replace(/\n*$/, "\n");
}
`

func buildOhMyPiScript(baseURL, key string, availableModels []string) string {
	providerBlock := []string{
		"  " + newapiProviderID + ":",
		"    baseUrl: " + yamlString(baseURL),
		"    api: openai-completions",
		"    apiKey: " + yamlString(key),
	}
	if len(availableModels) == 0 {
		providerBlock = append(providerBlock, "    models: []")
	} else {
		providerBlock = append(providerBlock, "    models:")
		for _, m := range availableModels {
			providerBlock = append(providerBlock, "      - id: "+yamlString(m))
		}
	}
	providerBlockJSON, _ := common.Marshal(providerBlock)

	var b strings.Builder
	b.WriteString("const fs = require(\"fs\");\nconst os = require(\"os\");\nconst path = require(\"path\");\n\n")
	b.WriteString(yamlProvidersEditorJS)
	b.WriteString(fmt.Sprintf(`
const configPath = %s;
fs.mkdirSync(path.dirname(configPath), { recursive: true });
let existing = "";
if (fs.existsSync(configPath)) {
  existing = fs.readFileSync(configPath, "utf8");
}
let output;
try {
  output = editProvidersYaml(existing, %s, %s);
} catch (e) {
  console.error("Could not parse existing configuration file: " + configPath + " (" + e.message + ")");
  process.exit(1);
}
fs.writeFileSync(configPath, output, "utf8");
console.log("Oh My Pi configured: " + configPath);
`, jsHomedirPath(".omp", "agent", "models.yml"), jsString(newapiProviderID), string(providerBlockJSON)))
	return b.String()
}

// ---- Oh My Pi (YAML) — PowerShell / Windows --------------------------------
//
// A direct port of yamlProvidersEditorJS above. Get-LineSlice always writes
// its result via Write-Output -NoEnumerate so a single-element (or empty)
// slice is never unwrapped to a scalar/$null by PowerShell's pipeline —
// callers always get a real array back, however many lines it holds.
const psYamlProvidersEditor = `function Test-YamlSanity {
    param([string]$Text)
    if ($Text.IndexOf([char]9) -ge 0) {
        return "file contains tab characters, which is not valid YAML indentation"
    }
    return $null
}

function Test-TopLevelKeyLine {
    param([string]$Line)
    if ($Line -match '^[A-Za-z0-9_.-]+:\s*(#.*)?$') { return $true }
    if ($Line -match '^[A-Za-z0-9_.-]+:\s+\S.*$') { return $true }
    return $false
}

function Get-ProviderEntryId {
    param([string]$Line)
    $m = [regex]::Match($Line, '^  ([A-Za-z0-9_.-]+):\s*(#.*)?$')
    if ($m.Success) { return $m.Groups[1].Value }
    return $null
}

function Get-LineSlice {
    param($Lines, [int]$Start, [int]$EndExclusive)
    $result = New-Object System.Collections.Generic.List[string]
    for ($k = $Start; $k -lt $EndExclusive; $k++) {
        if ($k -ge 0 -and $k -lt $Lines.Length) { $result.Add([string]$Lines[$k]) }
    }
    Write-Output -NoEnumerate $result.ToArray()
}

function Edit-ProvidersYaml {
    param(
        [string]$ExistingText,
        [string]$ProviderId,
        [string[]]$ProviderBlockLines
    )
    $text = if ($ExistingText) { $ExistingText } else { '' }
    if ($text.Trim() -ne '') {
        $problem = Test-YamlSanity $text
        if ($problem) { throw $problem }
    }

    $linesArr = if ($text.Length -gt 0) { [regex]::Split($text, '\r\n|\n') } else { @() }

    $providersStart = -1
    $providersEnd = $linesArr.Length
    for ($i = 0; $i -lt $linesArr.Length; $i++) {
        if ($linesArr[$i] -eq 'providers:') {
            $providersStart = $i
            $providersEnd = $linesArr.Length
            for ($j = $i + 1; $j -lt $linesArr.Length; $j++) {
                if (Test-TopLevelKeyLine $linesArr[$j]) {
                    $providersEnd = $j
                    break
                }
            }
            break
        }
    }

    if ($providersStart -eq -1) {
        $before = $linesArr
        $after = @()
        $providersBody = @()
    } else {
        $before = Get-LineSlice $linesArr 0 $providersStart
        $after = Get-LineSlice $linesArr $providersEnd $linesArr.Length
        $providersBody = Get-LineSlice $linesArr ($providersStart + 1) $providersEnd
    }

    $entries = New-Object System.Collections.Generic.List[object]
    $i = 0
    while ($i -lt $providersBody.Length) {
        $id = Get-ProviderEntryId $providersBody[$i]
        if ($id) {
            $j = $i + 1
            while ($j -lt $providersBody.Length -and (-not (Get-ProviderEntryId $providersBody[$j])) -and ($providersBody[$j].StartsWith('   ') -or $providersBody[$j].Trim() -eq '')) {
                $j++
            }
            $entries.Add([PSCustomObject]@{ Id = $id; Start = $i; End = $j })
            $i = $j
        } else {
            $i++
        }
    }

    $replaced = $false
    $newBody = New-Object System.Collections.Generic.List[string]
    $cursor = 0
    foreach ($entry in $entries) {
        foreach ($l in (Get-LineSlice $providersBody $cursor $entry.Start)) { $newBody.Add($l) }
        if ($entry.Id -eq $ProviderId) {
            foreach ($l in $ProviderBlockLines) { $newBody.Add($l) }
            $replaced = $true
        } else {
            foreach ($l in (Get-LineSlice $providersBody $entry.Start $entry.End)) { $newBody.Add($l) }
        }
        $cursor = $entry.End
    }
    foreach ($l in (Get-LineSlice $providersBody $cursor $providersBody.Length)) { $newBody.Add($l) }
    if (-not $replaced) {
        foreach ($l in $ProviderBlockLines) { $newBody.Add($l) }
    }

    $out = New-Object System.Collections.Generic.List[string]
    foreach ($l in $before) { $out.Add($l) }
    if ($before.Length -gt 0 -and $before[$before.Length - 1].Trim() -ne '') { $out.Add('') }
    $out.Add('providers:')
    foreach ($l in $newBody) { $out.Add($l) }
    foreach ($l in $after) { $out.Add($l) }

    while ($out.Count -gt 1 -and $out[$out.Count - 1].Trim() -eq '' -and $out[$out.Count - 2].Trim() -eq '') {
        $out.RemoveAt($out.Count - 1)
    }

    $joined = ($out -join [string][char]10)
    $joined = [regex]::Replace($joined, '\n*$', [string][char]10)
    return $joined
}
`

func buildOhMyPiScriptPS(baseURL, key string, availableModels []string) string {
	providerBlock := []string{
		"  " + newapiProviderID + ":",
		"    baseUrl: " + yamlString(baseURL),
		"    api: openai-completions",
		"    apiKey: " + yamlString(key),
	}
	if len(availableModels) == 0 {
		providerBlock = append(providerBlock, "    models: []")
	} else {
		providerBlock = append(providerBlock, "    models:")
		for _, m := range availableModels {
			providerBlock = append(providerBlock, "      - id: "+yamlString(m))
		}
	}

	var b strings.Builder
	b.WriteString(psYamlProvidersEditor)
	b.WriteString("\n$configPath = " + psHomedirPath(".omp", "agent", "models.yml") + "\n")
	b.WriteString("$dir = Split-Path -Path $configPath -Parent\n")
	b.WriteString("[System.IO.Directory]::CreateDirectory($dir) | Out-Null\n")
	b.WriteString("$existing = ''\n")
	b.WriteString("if (Test-Path -LiteralPath $configPath) {\n    $existing = [System.IO.File]::ReadAllText($configPath)\n}\n")
	b.WriteString("try {\n")
	b.WriteString("    $output = Edit-ProvidersYaml -ExistingText $existing -ProviderId " + psSingleQuote(newapiProviderID) +
		" -ProviderBlockLines " + psStringArrayLiteral(providerBlock) + "\n")
	b.WriteString("} catch {\n")
	b.WriteString("    [Console]::Error.WriteLine(\"Could not parse existing configuration file: \" + $configPath + \" (\" + $_.Exception.Message + \")\")\n")
	b.WriteString("    exit 1\n")
	b.WriteString("}\n")
	b.WriteString("$utf8NoBom = New-Object System.Text.UTF8Encoding($false)\n")
	b.WriteString("[System.IO.File]::WriteAllText($configPath, $output, $utf8NoBom)\n")
	b.WriteString("Write-Host (\"Oh My Pi configured: \" + $configPath)\n")
	return b.String()
}

// ---- outer OS wrappers ------------------------------------------------------

// setupScriptEnvVar describes the single environment variable Codex's script
// persists. No other application's script uses this.
type setupScriptEnvVar struct {
	Name    string
	Value   string
	Comment string
}

// renderPowerShellScript wraps a native PowerShell merge payload (produced by
// one of the build*ScriptPS functions above) in the outer script that sets
// strict error handling and, for Codex, persists the one required
// environment variable. Unlike the POSIX path, this has no Node.js (or any
// other) runtime dependency — see design.md — Decisions.
func renderPowerShellScript(psPayload string, envVar *setupScriptEnvVar) string {
	var b strings.Builder
	b.WriteString("$ErrorActionPreference = \"Stop\"\n")
	b.WriteString(psPayload)
	if !strings.HasSuffix(psPayload, "\n") {
		b.WriteString("\n")
	}
	if envVar != nil {
		b.WriteString(fmt.Sprintf("# %s\n", envVar.Comment))
		b.WriteString(fmt.Sprintf("[Environment]::SetEnvironmentVariable(%s, %s, \"User\")\n", psSingleQuote(envVar.Name), psSingleQuote(envVar.Value)))
		b.WriteString(fmt.Sprintf("$env:%s = %s\n", envVar.Name, psSingleQuote(envVar.Value)))
		b.WriteString(fmt.Sprintf("Write-Host \"Set %s for your user account. Restart your terminal for new sessions to see it.\"\n", envVar.Name))
	}
	b.WriteString("Write-Host \"Done.\"\n")
	return b.String()
}

// renderPosixScript wraps a Node.js payload (produced by one of the
// build*Script functions above) in a POSIX shell script. It checks for
// `node` on PATH before doing anything else and exits with a clear,
// actionable message if it is missing, changing nothing on disk (design.md —
// Decisions: the POSIX path keeps its Node.js dependency but must fail
// loudly and early rather than leaving the user with a bare
// "node: command not found").
//
// The payload is written to a temp file via a quoted heredoc (so it is never
// subject to shell expansion, regardless of its contents) and removed after
// running. mktemp's template ends in the literal "XXXXXX" with no further
// suffix: BSD/macOS mktemp only substitutes trailing X's, so a suffix like
// ".js" after them is not portable — Node executes a script by path
// regardless of its extension, so no suffix is needed.
func renderPosixScript(jsPayload string, envVar *setupScriptEnvVar) string {
	var b strings.Builder
	b.WriteString("#!/bin/sh\nset -e\n")
	b.WriteString("if ! command -v node >/dev/null 2>&1; then\n")
	b.WriteString("  echo \"Error: Node.js is required to run this configuration step but was not found on PATH. Install Node.js (https://nodejs.org/) and re-run this command.\" >&2\n")
	b.WriteString("  exit 1\n")
	b.WriteString("fi\n")
	b.WriteString("tmp=$(mktemp \"${TMPDIR:-/tmp}/newapi-setup-XXXXXX\")\n")
	b.WriteString("cat <<'NEWAPI_SETUP_JS_EOF' > \"$tmp\"\n")
	b.WriteString(jsPayload)
	if !strings.HasSuffix(jsPayload, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("NEWAPI_SETUP_JS_EOF\n")
	b.WriteString("node \"$tmp\"\nrm -f \"$tmp\"\n")
	if envVar != nil {
		exportLine := fmt.Sprintf("export %s=%s", envVar.Name, posixDoubleQuoted(envVar.Value))
		b.WriteString(fmt.Sprintf("# %s\n", envVar.Comment))
		b.WriteString("RC_FILE=\"$HOME/.profile\"\n")
		b.WriteString("case \"$SHELL\" in\n  */zsh) RC_FILE=\"$HOME/.zshrc\" ;;\n  */bash) RC_FILE=\"$HOME/.bashrc\" ;;\nesac\n")
		b.WriteString("touch \"$RC_FILE\"\n")
		// grep exits 1 when the pattern simply has no match (the common case
		// on a first run) — that must not be treated as failure. Only exit
		// status >= 2 (a genuine read/write error) aborts.
		b.WriteString(fmt.Sprintf("if grep -v '^export %s=' \"$RC_FILE\" > \"$RC_FILE.newapi_tmp\" 2>/dev/null; then\n  grep_status=0\nelse\n  grep_status=$?\nfi\n", envVar.Name))
		b.WriteString("if [ \"$grep_status\" -gt 1 ]; then\n")
		b.WriteString("  echo \"Error: failed to update $RC_FILE (grep exited with status $grep_status); left unchanged.\" >&2\n")
		b.WriteString("  rm -f \"$RC_FILE.newapi_tmp\"\n")
		b.WriteString("  exit 1\n")
		b.WriteString("fi\n")
		// An empty filtered result over a non-empty RC_FILE is a second,
		// independent safety net against a failed or truncated write — but it
		// is also the *expected* outcome when every line in RC_FILE was
		// already one of our own owned "export NEWAPI_API_KEY=" lines (e.g.
		// re-running this script against an rc file it created earlier).
		// Distinguish the two: only abort when RC_FILE actually contains a
		// line that is not an owned export line, since that means the
		// filtered-empty result cannot be explained by legitimate filtering.
		b.WriteString("if [ -s \"$RC_FILE\" ] && [ ! -s \"$RC_FILE.newapi_tmp\" ]; then\n")
		b.WriteString(fmt.Sprintf("  if grep -vq '^export %s=' \"$RC_FILE\" 2>/dev/null; then\n    non_owned_status=0\n  else\n    non_owned_status=$?\n  fi\n", envVar.Name))
		b.WriteString("  if [ \"$non_owned_status\" -ne 1 ]; then\n")
		b.WriteString("    echo \"Error: refusing to replace $RC_FILE with an empty file; left unchanged.\" >&2\n")
		b.WriteString("    rm -f \"$RC_FILE.newapi_tmp\"\n")
		b.WriteString("    exit 1\n")
		b.WriteString("  fi\n")
		b.WriteString("fi\n")
		b.WriteString("mv \"$RC_FILE.newapi_tmp\" \"$RC_FILE\"\n")
		b.WriteString(fmt.Sprintf("echo %s >> \"$RC_FILE\"\n", posixSingleQuote(exportLine)))
		b.WriteString(exportLine + "\n")
		b.WriteString(fmt.Sprintf("echo \"Set %s in $RC_FILE and the current shell. Restart your shell (or re-source $RC_FILE) for new shells to see it.\"\n", envVar.Name))
	}
	b.WriteString("echo \"Done.\"\n")
	return b.String()
}

// ---- handler ----------------------------------------------------------------

// GetTokenSetupScript is the public, unauthenticated GET /api/setup/script
// handler (registered at that path in router/api-router.go; the Go name
// stays distinct from the existing GetSetup system-installation handler).
// It is authenticated only by the supplied key: the endpoint resolves that
// key's effective group, validates every requested model against it, and
// returns the rendered script as text/plain.
func GetTokenSetupScript(c *gin.Context) {
	appInfo, ok := setupScriptApps[setupScriptApp(c.Query("app"))]
	if !ok {
		c.String(http.StatusBadRequest, common.TranslateMessage(c, i18n.MsgSetupScriptUnknownApp))
		return
	}
	targetOS, ok := parseSetupScriptOS(c.Query("os"))
	if !ok {
		c.String(http.StatusBadRequest, common.TranslateMessage(c, i18n.MsgSetupScriptUnknownOS))
		return
	}

	key := normalizeTokenKey(c.Query("key"))
	if key == "" {
		c.String(http.StatusBadRequest, common.TranslateMessage(c, i18n.MsgTokenNotProvided))
		return
	}

	token, err := model.GetTokenByKey(key, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.String(http.StatusOK, common.TranslateMessage(c, i18n.MsgTokenInvalid))
			return
		}
		common.SysError("failed to get token for public setup script: " + err.Error())
		c.String(http.StatusInternalServerError, common.TranslateMessage(c, i18n.MsgTokenGetInfoFailed))
		return
	}

	group, err := resolveEffectiveTokenGroup(token)
	if err != nil {
		common.SysError("failed to resolve effective group for public setup script: " + err.Error())
		c.String(http.StatusInternalServerError, common.TranslateMessage(c, i18n.MsgTokenGetInfoFailed))
		return
	}
	availableModels := model.GetGroupEnabledModels(group)
	availableSet := make(map[string]bool, len(availableModels))
	for _, m := range availableModels {
		availableSet[m] = true
	}

	appName := setupScriptApp(c.Query("app"))
	models := make(map[string]string, len(appInfo.Slots))
	for _, slot := range appInfo.Slots {
		value := strings.TrimSpace(c.Query("model_" + slot))
		if value == "" {
			c.String(http.StatusBadRequest, common.TranslateMessage(c, i18n.MsgSetupScriptSlotRequired, map[string]any{"Slot": slot}))
			return
		}
		if !availableSet[value] {
			c.String(http.StatusBadRequest, common.TranslateMessage(c, i18n.MsgSetupScriptModelNotAllowed, map[string]any{"Model": value}))
			return
		}
		models[slot] = value
	}

	// Fall back to the request's own scheme+host when an operator has never
	// configured system_setting.ServerAddress, so the generated script never
	// embeds an empty base URL (the frontend applies the same
	// window.location.origin fallback for display).
	baseURL := strings.TrimRight(system_setting.ServerAddress, "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(requestOrigin(c), "/")
	}

	var envVar *setupScriptEnvVar
	if appName == setupScriptAppCodex {
		envVar = &setupScriptEnvVar{
			Name:    "NEWAPI_API_KEY",
			Value:   token.Key,
			Comment: "Codex reads the API key only from this environment variable; there is no documented way to store it directly in config.toml.",
		}
	}

	var script string
	switch targetOS {
	case setupScriptOSWindows:
		var psPayload string
		switch appName {
		case setupScriptAppClaudeCode:
			psPayload = buildClaudeCodeScriptPS(baseURL, token.Key, models)
		case setupScriptAppCodex:
			psPayload = buildCodexScriptPS(baseURL, token.Key, models)
		case setupScriptAppOpenCode:
			psPayload = buildOpenCodeScriptPS(baseURL, token.Key, models, availableModels)
		case setupScriptAppPi:
			psPayload = buildPiScriptPS(baseURL, token.Key, availableModels)
		case setupScriptAppOhMyPi:
			psPayload = buildOhMyPiScriptPS(baseURL, token.Key, availableModels)
		}
		script = renderPowerShellScript(psPayload, envVar)
	case setupScriptOSUnix:
		var jsPayload string
		switch appName {
		case setupScriptAppClaudeCode:
			jsPayload = buildClaudeCodeScript(baseURL, token.Key, models)
		case setupScriptAppCodex:
			jsPayload = buildCodexScript(baseURL, token.Key, models)
		case setupScriptAppOpenCode:
			jsPayload = buildOpenCodeScript(baseURL, token.Key, models, availableModels)
		case setupScriptAppPi:
			jsPayload = buildPiScript(baseURL, token.Key, availableModels)
		case setupScriptAppOhMyPi:
			jsPayload = buildOhMyPiScript(baseURL, token.Key, availableModels)
		}
		script = renderPosixScript(jsPayload, envVar)
	}

	c.String(http.StatusOK, script)
}
