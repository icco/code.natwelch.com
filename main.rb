# frozen_string_literal: true

# What user we care about.
USER = "icco"

class Code < Sinatra::Base
  Google::Cloud.configure do |config|
    config.trace.capture_stack = true
    config.service_name = "code"
  end

  use Google::Cloud::Logging::Middleware
  use Google::Cloud::ErrorReporting::Middleware
  use Google::Cloud::Trace::Middleware

  ActiveRecord::Base.include_root_in_json = true
  ActiveRecord::Base.store_full_sti_class = true
  ActiveSupport.use_standard_json_time_format = true
  ActiveSupport.escape_html_entities_in_json = false
  ActiveRecord::Base.default_timezone = :utc

  # Now we can estabilish connection with our db
  url = URI(ENV["DATABASE_URL"])
  options = {
    host: url.host,
    port: url.port,
    database: url.path[1..],
    username: url.user,
    password: url.password,
    adapter: "postgresql"
  }
  ActiveRecord::Base.establish_connection(options)

  use ConnectionPoolManagement
  enable :sessions
  register SassInitializer

  set :session_secret, ENV["SESSION_SECRET"] || "blargh"
  set :protection, except: :path_traversal
  set :protect_from_csrf, true

  get "/" do
    erb :index
  end

  get "/healthz" do
    "ok"
  end

  post "/save" do
    payload = JSON.parse(request.body.read).symbolize_keys

    Commit.factory payload[:user], payload[:repo], payload[:sha], nil, true
  end

  get "/data/commit.csv" do
    logger.info "USER is #{USER.inspect}."
    data = Commit.order(:created_on).where(user: USER).where("created_on >= ?",
                                                             Chronic.parse("2009-01-01")).group(:created_on).count

    @stats = Hash.new(0)
    data.each do |row|
      @stats[row[0].strftime("%D")] += row[1]
    end

    etag "data/commit-#{Commit.maximum(:created_on)}"
    content_type "text/csv"
    erb :"commit_data.csv"
  end

  get "/data/:year/weekly.csv" do
    @year = params[:year] || Time.now.year.to_s
    logger.info "Getting data for #{@year}."

    logger.info "USER is #{USER.inspect}."
    data = Commit.order(:created_on).where(user: USER).group(:created_on).count

    @stats = Hash.new(0)
    ("01".."52").each { |week| @stats[week] = 0 }
    data.each do |row|
      if row[0].strftime("%Y") == @year
        week = row[0].strftime("%U")
        @stats[week] += row[1] if week != "00"
      end
    end

    etag "data/weekly-#{@year}-#{Commit.maximum(:created_on)}"
    content_type "text/csv"
    erb :"weekly_data.csv"
  end
end

def new_client
  options = { auto_paginate: true }
  if ENV["GITHUB_CLIENT_ID"]
    options[:client_id] = ENV["GITHUB_CLIENT_ID"]
    options[:client_secret] = ENV["GITHUB_CLIENT_SECRET"]
    options[:netrc] = false
  end

  Octokit::Client.new(options)
end

# Gets repos for user and all of their public orgs.
def user_repos(user_name, client)
  if client.ratelimit.remaining <= 2
    raise "Github ratelimit remaining #{client.ratelimit.remaining} of #{client.ratelimit.limit} is not enough."
  end

  logger.info "Looking up repos for #{user_name.inspect}."
  repos = client.repos(user_name).map { |r| r["full_name"].split("/") }
  client.orgs(user_name).each do |org|
    logger.info "Adding #{org['login']} repos."
    repos.concat(client.org_repos(org["login"]).map { |r| r["full_name"].split("/") })
  end
  logger.info "Found #{repos.count} for #{user_name.inspect}."

  repos
end
