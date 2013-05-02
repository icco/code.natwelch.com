require 'padrino-core/cli/rake'

require File.expand_path('../config/boot.rb', __FILE__)

PadrinoTasks.use(:database)
PadrinoTasks.use(:activerecord)
PadrinoTasks.init


# Adds extended DateTime functionality
require "date"

PadrinoTasks.init

desc "Run a local server."
task :local do
  Kernel.exec("shotgun -s thin -p 9393")
end

desc "Run a local server."
task :local do
  Kernel.exec("bundle exec shotgun -s thin")
end

desc "Runs all of the tasks that store data."
task :cron => [ "cron:hourly" ]

task :stats do
  puts "Commits by #{USER}:\t#{Commit.filter(:user => USER).count}"
  puts "Github Ratelimit:\t#{Octokit.ratelimit_remaining}/#{Octokit.ratelimit}"
end

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
      logger.push "Inserted for #{year}-#{month}-#{day}, #{hour}:00", :info
    end
  end

  desc "Loops through every hour this year, and puts it all into the db."
  task :rebuild do
    # githubarchive started 3/11/2012
    start = Chronic.parse "March 11, 2012"
    finish = Time.now

    # We can't iterate over time in Ruby (Time#succ is crazy expensive), so
    # instead we use a do-while loop.
    time = start
    begin
      print "."
      Commit.fetchAllForTime time.day, time.month, time.year, time.hour
    end while (time += 3600) < finish
  end

  desc "gets all of the commits from every public repo of USER."
  task :get_older_commits do

    # For testing forked repos.
    # commits = Octokit.commits("icco/downforeveryoneorjustme").delete_if {|commit| commit.is_a? String }
    # commits.each do |commit|
    #   p Commit.factory "icco", "downforeveryoneorjustme", commit['sha']
    # end

    # Now, because we will probably want some data from before when github
    # archive started, lets pound github's api and get some older commits.
    #
    # NOTE: If you have a lot of repos or lots of commits, you could blow out
    # your request quota from github. Remove the auto_traveral from the commits
    # call if this is the case.
    client = Octokit::Client.new({:auto_traversal => true})
    client.repos(USER).each do |repo|
      puts "#{USER}/#{repo["name"]}"
      commits = client.commits("#{USER}/#{repo["name"]}").delete_if {|commit| commit.is_a? String }
      commits.each do |commit|
        p Commit.factory USER, repo['name'], commit['sha']
      end
    end
  end
end

namespace :db do

  desc "Erase and Rebuild the Database."
  task :rebuild => [ 'db:erase', 'cron:rebuild', 'cron:get_older_commits' ]

end
