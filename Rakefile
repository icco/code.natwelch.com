require "rubygems"
require "bundler"

Bundler.require

require "./models"

# Adds extended DateTime functionality
require "date"

USER = "icco"

desc "Run a local server."
task :local do
  Kernel.exec("bundle exec shotgun -s thin")
end

desc "Runs all of the tasks that store data."
task :cron => [ "cron:repositories", "cron:commits"]

namespace :cron do

  desc "Goes through the events of yesterday and puts them in the db."
  task :hourly do
    time = Chronic.parse "yesterday"
    day = time.day
    month = time.month
    year = time.year

    hours = 0..23

    hours.each do |hour|
      Commit.fetchAllForTime day, month, year, hour
    end
  end

  desc "Loops through every hour this year, and puts it all into the db."
  task :rebuild do
    year = 2012
    (1..12).each do |month|
      (0..31).each do |day|
        (0..23).each do |hour|
          Commit.fetchAllForTime day, month, year, hour
        end
      end
    end
  end
end

namespace :db do

  desc "Bring database schema up to par."
  task :migrate do
    db_url = ENV["DATABASE_URL"] || "sqlite://db/data.db"
    migrations_dir = "./db/migrations/"

    puts "Migrating from "#{migrations_dir}" into "#{db_url}"."

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
    DB = Sequel.connect(ENV["DATABASE_URL"] || "sqlite://db/data.db")
    DB.drop_table(:commits)
  end

  desc "Dumps the database"
  task :dump do
    DB = Sequel.connect(ENV["DATABASE_URL"] || "sqlite://db/data.db")

    puts "Sites Schema"
    p DB.schema :sites

    puts "Commits Schema"
    p DB.schema :commits
  end
end
