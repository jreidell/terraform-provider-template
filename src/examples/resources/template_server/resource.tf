resource "template_server" "test" {
  uuid_count = 5

  # Optional delays, in seconds, that simulate a slow API.
  creation_delay = 5
  update_delay   = 3
  deletion_delay = 2
}

output "generated_uuids" {
  value = template_server.test.uuids
}
