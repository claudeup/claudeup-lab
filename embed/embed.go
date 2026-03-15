package assets

import _ "embed"

//go:embed Dockerfile
var Dockerfile []byte

//go:embed devcontainer.template.json
var DevcontainerTemplate string

//go:embed features.json
var FeaturesJSON []byte

//go:embed init-claude-config.sh
var InitClaudeConfig []byte

//go:embed init-config-repo.sh
var InitConfigRepo []byte

//go:embed init-claudeup.sh
var InitClaudeup []byte
