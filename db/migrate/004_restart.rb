class Restart < ActiveRecord::Migration
  def change
    # This essentially defeats the purpose of having a migration.
    drop_table :commits
    drop_table :entries
    drop_table :repos

    create_table :commits do |t|
      t.string :repo
      t.string :user
      t.string :sha
      t.datetime "created_on"
    end
  end
end
