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

get "/data/commit.csv" do
  data = Commit.group_and_count(:created_on)

  @stats = Hash.new(0)
  data.each do |row|
    @stats[row.values[:created_on].strftime("%D")] += row.values[:count]
  end

  etag "data/commit-#{Commit.max(:created_on)}"
  content_type "text/csv"
  erb :"commit_data.csv"
end

get "/style.css" do
  content_type "text/css", :charset => "utf-8"
  less :style
end
