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

    # Install dependencies:
    sudo apt-get install jq mysql-client

    # To install it in BASH:
 
    echo 'navisql() {' >> ~/.bashrc
    echo '  ~/navisql/navisql "$@"' >> ~/.bashrc
    echo '}' >> ~/.bashrc
    echo 'navi() {' >> ~/.bashrc
    echo '  ~/navisql/navi "$@"' >> ~/.bashrc
    echo '}' >> ~/.bashrc
    echo 'source ~/navisql/navisql_autocomplete' >> ~/.bashrc

    # To install it in ZSH:

    echo 'navisql() {' >> ~/.zshrc
    echo '  ~/navisql/navisql "$@"' >> ~/.zshrc
    echo '}' >> ~/.zshrc
    echo 'navi() {' >> ~/.zshrc
    echo '  ~/navisql/navi "$@"' >> ~/.zshrc
    echo '}' >> ~/.zshrc
    echo 'source ~/navisql/navisql_autocomplete' >> ~/.zshrc

    # Restart your shell:
    bash
    # or
    zsh 

Configure it:

    # navisql add-connection <connection_name> <user> <password> [<host> [<port>]]
    navisql add-connection local sebas asfdasfd

    # navisql cache-build <connection>
    # will retrieve the list of databases and tables for the given connection
    # and store it in a cache file to be used in the auto-completion
    navisql cache-build local

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

Note that most of the commands are auto-completed, so you can just type the first few letters and press `TAB` to auto-complete, including database names and table names once you run the cache-build utility for that database.  Also field names will be autocompleted, retrieving the field names from the table schema.

    $ navisql show local my_project_database users 1

    - id: 1
    - name: sebas
    - email: sebas@mydomain.com.ar
    - password: $2y$10$jAvUE0y123412341234123412341234
    - api_token: 12341234123412341234123412341234
    - google2fa_secret: L1234123412341234
    - remember_token: 12341234123412341234123412341234
    - user_group_id: 1 [navi 1]
    - type: 1
    - user_status: 1
    - language: en
    - user_timezone_id: 1 [navi 2]
    - created_at: 1540232836
    - updated_at: 1715020282
    - deleted_at: NULL
    - email_verified_at: NULL
      [ Web edit: [navi 3] http://www.local.ip1.cc/adminer/?server=127.0.0.1&username=sebas&db=my_project_database&edit=users&where%5Bid%5D=1 ]



    # let's now type the suggested command to explore the user_group_id:

    $ navi 1
    show local my_project_database user_groups 1 
    - id: 1 
    - name: Root 
    - type: Manager 
    - description: Main group
    - secret: asecretword1234 
    - status: 1 
    - currency_id: 1 [navi 4]
    - balance_id: 0 [navi 5]
    - deleted_at: NULL 
    [ Web edit: [navi 6]  http://www.local.ip1.cc/adminer/?server=127.0.0.1&username=sebas&db=my_project_database&edit=user_groups&where%5Bid%5D=1 ]

