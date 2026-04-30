terraform {
  backend "gcs" {
    prefix = "lume/prod"
  }
}

