# What user we care about.
USER = "icco"
logger = Logger.new(STDOUT)

class Code < Sinatra::Base
  use ConnectionPoolManagement
  enable :sessions
  register SassInitializer

  set :session_secret, ENV['SESSION_SECRET'] || 'blargh'
  set :protection, :except => :path_traversal
  set :protect_from_csrf, true

  get "/" do
    erb :index
  end

  get "/healthz" do
    "ok"
  end


  get "/cron" do
    time = Chronic.parse "yesterday"
    day = time.day
    month = time.month
    year = time.year

    client = new_client
    (0..23).each do |hour|
      Commit.fetchAllForTime day, month, year, hour, client
      logger.info "Inserted for #{year}-#{month}-#{day}, #{hour}:00"
    end

    user_repos(USER, client).sample(10).each do |repo|
      logger.info "#{repo[0]}/#{repo[1]}"
      Commit.update_repo repo[0], repo[1], client, true
    end
  end

  get "/data/commit.csv" do
    logger.info "USER is #{USER.inspect}."
    data = Commit.order(:created_on).where(:user => USER).where("created_on >= ?", Chronic.parse("2009-01-01")).group(:created_on).count()

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
    data = Commit.order(:created_on).where(:user => USER).group(:created_on).count()

    @stats = Hash.new(0)
    ("01".."52").each {|week| @stats[week] = 0 }
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
