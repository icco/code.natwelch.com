Sequel.migration do
  change do
    create_table :entries do
      primary_key :id
      String :user
      Integer :repos, :default => 0
      Integer :forks, :default => 0
      Integer :watchers, :default => 0
      DateTime :created_on
    end
  end
end
