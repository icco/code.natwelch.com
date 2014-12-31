require 'padrino-core/cli/rake'

require File.expand_path('../config/boot.rb', __FILE__)

PadrinoTasks.use(:database)
PadrinoTasks.use(:activerecord)
PadrinoTasks.init

# Adds extended DateTime functionality
require "date"

PadrinoTasks.init

def new_client
  options = {:auto_paginate => true}
  if ENV['GITHUB_CLIENT_ID']
    options[:client_id] = ENV['GITHUB_CLIENT_ID']
    options[:client_secret] = ENV['GITHUB_CLIENT_SECRET']
    options[:netrc] = false
  end
  client = Octokit::Client.new(options)

  return client
end

# Gets repos for user and all of their public orgs.
def user_repos user_name, client
  raise "Github ratelimit remaining #{client.ratelimit.remaining} of #{client.ratelimit.limit} is not enough." if client.ratelimit.remaining <= 2
  logger.info "Looking up repos for #{user_name.inspect}."
  repos = client.repos(user_name).map {|r| r["full_name"].split("/") }
  client.orgs(user_name).each do |org|
    logger.info "Adding #{org["login"]} repos."
    repos = repos.concat(client.org_repos(org["login"]).map {|r| r["full_name"].split("/") })
  end
  logger.info "Found #{repos.count} for #{user_name.inspect}."

  return repos
end

desc "Run a local server."
task :local do
  Kernel.exec("shotgun -s thin -p 9393")
end

desc "Run a local server."
task :local do
  Kernel.exec("bundle exec shotgun -s thin")
end

desc "Print github request stats."
task :stats do
  client = new_client
  puts "Commits by #{USER}:\t#{Commit.where(:user => USER).count}"
  puts "Github Ratelimit:\t#{client.ratelimit.remaining}/#{client.ratelimit.limit}"
end

desc "Ping code.natwelch.com."
task :ping do
  require "open-uri"
  uri = URI.parse "http://code.natwelch.com"
  uri.open do |data|
    logger.info "Pinging code.natwelch.com. Headers: #{data.meta.inspect}"
  end
end

desc "Dump user info for a user."
task :print_user do
  client = new_client
  user = Commit.lookup_user "nat@natwelch.com"
  p client.user user
end

namespace :history do

  desc "Loops through every hour, and puts it all into the db."
  task :rebuild do
    # githubarchive started 3/11/2012
    start = Chronic.parse "March 11, 2012"
    finish = Time.now

    # We can't iterate over time in Ruby (Time#succ is crazy expensive), so
    # instead we use a do-while loop.
    time = start
    begin
      Commit.fetchAllForTime time.day, time.month, time.year, time.hour, new_client
    end while (time += 3600) < finish
  end

  desc "Loops through every hour of 2014, and puts it all into the db."
  task :rebuild_2014 do
    # githubarchive started 3/11/2012
    start = Chronic.parse "January 1, 2014"
    finish = Time.now
    client = new_client

    # We can't iterate over time in Ruby (Time#succ is crazy expensive), so
    # instead we use a do-while loop.
    time = start
    begin
      Commit.fetchAllForTime time.day, time.month, time.year, time.hour, client
    end while (time += 3600) < finish
  end

  desc "Gets all of the commits from every public repo of USER."
  task :get_older_commits do
    # Now, because we will probably want some data from before when github
    # archive started, lets pound github's api and get some older commits.
    client = new_client
    user_repos(USER, client).each do |repo|
      logger.info "#{repo[0]}/#{repo[1]}"
      Commit.update_repo repo[0], repo[1], client, false
    end
  end

  desc "Dumps all of a user and his org's repos."
  task :dump_repos do
    client = new_client
    user_repos(USER, client).each do |repo|
      logger.info repo.inspect
    end
  end
end

namespace :cron do

  desc "Goes through the events of yesterday and puts them in the db."
  task :hourly do
    time = Chronic.parse "yesterday"
    day = time.day
    month = time.month
    year = time.year

    hours = 0..23
    client = new_client

    hours.each do |hour|
      Commit.fetchAllForTime day, month, year, hour, client
      logger.push "Inserted for #{year}-#{month}-#{day}, #{hour}:00", :info
    end
  end

  desc "Gets all of the commits from 10 random repos of USER."
  task :daily do
    logger.info "USER is #{USER.inspect}."
    client = new_client
    user_repos(USER, client).sample(10).each do |repo|
      logger.info "#{repo[0]}/#{repo[1]}"
      Commit.update_repo repo[0], repo[1], client, true
    end
  end
end
