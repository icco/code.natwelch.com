require 'rubygems'
require 'bundler'

Bundler.require

require 'models'

desc "Writes current repo counts to db."
task :cron do
  user_name = "icco"
  herder = OctocatHerder.new
  me = herder.user user_name

  me.repositories.each do |repo|
    #puts "#{repo.name}\t#{repo.forks}\t#{repo.watchers}"
    r = Repo.new
    r.user = user_name
    r.repo = repo.name
    r.watchers = repo.watchers
    r.forks = repo.forks
    r.created_on = Time.now
    r.save
  end
end

namespace :db do

  desc "Bring database schema up to par."
  task :migrate do
    db_url = ENV['DATABASE_URL'] || "sqlite://db/data.db"
    migrations_dir = "./db/migrations/"

    puts "Migrating from '#{migrations_dir}' into '#{db_url}'."

    ret = Kernel.system("sequel -m #{migrations_dir} #{db_url}");

    if ret
      puts "Database migrated."
    else
      puts "Database migration failed."
    end

    puts "Database built."
  end

  desc "Delete the database"
  task :erase do
    DB = Sequel.connect(ENV['DATABASE_URL'] || 'sqlite://db/data.db')
    DB.drop_table(:sites)
    DB.drop_table(:commits)
    DB.drop_table(:schema_info)
  end

  desc "Dumps the database"
  task :dump do
    DB = Sequel.connect(ENV['DATABASE_URL'] || 'sqlite://db/data.db')

    puts "Sites Schema"
    p DB.schema :sites

    puts "Commits Schema"
    p DB.schema :commits
  end
end
