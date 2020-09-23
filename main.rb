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
