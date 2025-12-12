storage "file" {
  path = "/vault/data"
}

listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = 1
}

api_addr = "http://0.0.0.0:8200"
ui = true
ui = true

# Enable transit secrets engine on startup (requires initialization)
# This must be done manually after first start:
# vault secrets enable transit
