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

get "/data.csv" do
  @stats = StatEntry.all

  etag "data-#{StatEntry.max(:created_on)}"
  content_type "text/csv"
  erb :"data.csv"
end

get "/style.css" do
  content_type "text/css", :charset => "utf-8"
  less :style
end
