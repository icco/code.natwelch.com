Code.controllers  do
  ONE_HOUR = 60*60
  SHOULD_CACHE = Padrino.env != :development

  get :index, :cache => false do
    logger.info "This is development." if Padrino.env == :development
    render :index
  end

  get :healthz, :cache => false do
    "ok"
  end

  get "/data/commit.csv", :cache => SHOULD_CACHE do
    expires ONE_HOUR

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

  get "/data/:year/weekly.csv", :cache => SHOULD_CACHE do
    expires ONE_HOUR

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
    render :"weekly_data.csv"
  end
end
