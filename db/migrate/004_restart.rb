# frozen_string_literal: true

class Restart < ActiveRecord::Migration
  def change
    # This essentially defeats the purpose of having a migration.
    drop_table :commits if ActiveRecord::Base.connection.table_exists? "commits"
    drop_table :entries if ActiveRecord::Base.connection.table_exists? "entries"
    drop_table :repos if ActiveRecord::Base.connection.table_exists? "repos"

    create_table :commits do |t|
      t.string :repo
      t.string :user
      t.string :sha
      t.datetime "created_on"
    end
  end
end
