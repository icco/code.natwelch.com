#!/usr/bin/env ruby
#
# @author Nat Welch - https://github.com/icco

require './models'

configure do
  set :sessions, true
end

get '/' do
  @repos = Repo.all
  erb :index, :locals => {}
end

get '/style.css' do
  content_type 'text/css', :charset => 'utf-8'
  less :style
end
