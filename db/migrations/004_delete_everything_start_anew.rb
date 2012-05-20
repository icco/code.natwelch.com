Sequel.migration do
  change do
    # This essentially defeats the purpose of having a migration.
    drop_table(:commits)
    drop_table(:entries)
    drop_table(:repos)

    create_table :commits do
      primary_key :id
      String :repo
      String :user
      String :sha

      DateTime :created_on
    end
  end
end
