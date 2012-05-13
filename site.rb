#!/usr/bin/env ruby
#
# @author Nat Welch - https://github.com/icco

require "./models"

configure do
  set :sessions, true
end

get "/" do
  erb :index
end

get "/data/repo.csv" do
  @stats = StatEntry.order(:created_on).all

  etag "data/repo-#{StatEntry.max(:created_on)}"
  content_type "text/csv"
  erb :"repo_data.csv"
end

get "/data/commits.csv" do
  @stats = CommitCount.order(:created_on).all

  etag "data/commit-#{CommitCount.max(:created_on)}"
  content_type "text/csv"
  erb :"commit_data.csv"
end

get "/style.css" do
  content_type "text/css", :charset => "utf-8"
  less :style
end
