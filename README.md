navisql
=======
A MYSQL Navigator
-----------------

This is a tiny but useful bash utility to navigate MySQL databases from the command line.

It is a wrapper around the `mysql` command line client, and it provides a few useful features: 

- **Auto-completion**: it auto-completes table names, column names, and SQL keywords.
- **FK References**: it helps you navigate into related records by showing references.

Requirements:
- `mysql` command line client
- `jq` command line JSON processor
  - (Install it with `sudo apt-get install jq`)

Sample usage:
-------------

Install it:

    git clone https://github.com/neitanod/navisql.git ~/navisql

    echo "alias navisql='~/navisql/navisql'" >> ~/.bashrc
    echo "source ~/navisql/navisql_autocomplete" >> ~/.bashrc

    echo "alias navisql='~/navisql/navisql'" >> ~/.zshrc
    echo "source ~/navisql/navisql_autocomplete" >> ~/.zshrc

    # Restart your shell or run this right away:

    alias navisql='~/navisql/navisql'
    source ~/navisql/navisql_autocomplete

    # Install dependencies:
    sudo apt-get install jq mysql-client

Configure it:

    # navisql add-connection <connection_name> <user> <password> [<host> [<port>]]
    navisql add-connection local sebas asfdasfd

Customize it:

    # navisql_add_fk <connection> <db> <table> <field> <referenced_table> [<referenced_field>]"

    # Example fk:     users.user_group_id references user_groups.id
    navisql add-fk local my_project_database users user_group_id user_groups

    # Example fk:     users.user_timezone_id references user_timezone.id
    navisql add-fk local my_project_database users user_timezone_id user_timezone

    # Add a web edit url template to the get links to the adminer tool
    # that you'll be able to ctrl+click to open in your browser:
    navisql configure add "web_edit" "http://www.local.ip1.cc/adminer/?server={{SERVER}}&username={{USER}}&db={{DB}}&edit={{TABLE}}&where%5Bid%5D={{ID}}"

Use it:

    $ navisql show local my_project_database users 1

    - id: 1
    - name: sebas
    - email: sebas@mydomain.com.ar
    - password: $2y$10$jAvUE0y123412341234123412341234
    - api_token: 12341234123412341234123412341234
    - google2fa_secret: L1234123412341234
    - remember_token: 12341234123412341234123412341234
    - user_group_id: 1
      navisql show local my_project_database user_groups 1
    - type: 1
    - user_status: 1
    - language: en
    - user_timezone_id: 1
      navisql show local my_project_database user_timezone 1
    - created_at: 1540232836
    - updated_at: 1715020282
    - deleted_at: NULL
    - email_verified_at: NULL
      [ Web edit: http://www.local.ip1.cc/adminer/?server=127.0.0.1&username=sebas&db=my_project_database&edit=users&where%5Bid%5D=1 ]

      # let's copy and paste the suggested command to explore the user_group_id:
      $ navisql show local my_project_database user_groups 1
