data "external_schema" "postgres" {
  program = [
    "go",
    "run",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "./model",
    "--dialect", "postgres",
  ]
}

env "postgres" {
  src = data.external_schema.postgres.url
  dev = "docker://postgres/15/dev"
  migration {
    dir = "file://migrations/postgres"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}

data "external_schema" "mysql" {
  program = [
    "go",
    "run",
    "ariga.io/atlas-provider-gorm",
    "load",
    "--path", "./model",
    "--dialect", "mysql",
  ]
}


env "mysql" {
  src = data.external_schema.mysql.url
  dev = "mysql://root:pass@localhost:3306/example"
  migration {
    dir = "file://migrations/mysql"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}
