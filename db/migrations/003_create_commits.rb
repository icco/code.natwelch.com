Sequel.migration do
  change do
    create_table :commits do
      primary_key :id
      String :repo
      String :user
      Integer :count
      DateTime :created_on
    end
  end
end
