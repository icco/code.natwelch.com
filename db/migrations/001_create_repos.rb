Sequel.migration do
  change do
    create_table :repos do
      primary_key :id
      String :repo
      String :user
      Integer :forks
      Integer :watchers
      DateTime :created_on
    end
  end
end