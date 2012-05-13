require 'rubygems'
require 'bundler'

Bundler.require

require './models'

USER = "icco"

desc "Run a local server."
task :local do
  Kernel.exec("bundle exec shotgun -s thin")
end

desc "Runs all of the tasks that store data."
task :cron => [ 'cron:repositories', 'cron:commits']

namespace :cron do

  desc "Writes current repo statistics to db."
  task :repositories do
    user_name = USER
    herder = OctocatHerder.new
    me = herder.user user_name

    stat = StatEntry.new
    stat.user = user_name
    stat.created_on = Time.now
    stat.save # save to init default values.

    me.repositories.each do |repo|
      r = Repo.factory(repo.name, user_name, repo.watchers, repo.forks)

      stat.repos += 1
      stat.watchers += r.watchers
      stat.forks += r.forks
    end

    p stat.save
  end

  desc "Writes commit data to db."
  task :commits do
    repos = Repo.getRepoNames USER

    repos.each do |repo|
      commits = Octokit.commits("#{USER}/#{repo}").delete_if do |commit|
        commit.is_a? String
      end

      dates = Hash.new(0)
      commits.each do |commit|
        dates[Time.new(commit.commit.author.date).strftime("%D")] += 1
      end

      dates.each_pair do |date,count|
        if count
          c = CommitCount.new
          c.created_on = Chronic.parse(date)
          c.count = count
          c.repo = repo
          c.user = USER
          c.save
        end
      end
    end
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
