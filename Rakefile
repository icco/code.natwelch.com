require 'rubygems'
require 'bundler'

Bundler.require

desc "Writes current repo counts to db."
task :cron do
  Octokit.repos("icco").each do |repo|
    puts "#{repo.name}\t#{repo.forks}\t#{repo.watchers}"
  end
end
