Code.controllers  do

  get :index do
    render :index
  end

  get "/data/commit.csv" do
    data = Commit.filter(:user => USER).group_and_count(:created_on).order(:created_on)

    @stats = Hash.new(0)
    data.each do |row|
      @stats[row.values[:created_on].strftime("%D")] += row.values[:count]
    end

    etag "data/commit-#{Commit.max(:created_on)}"
    content_type "text/csv"
    erb :"commit_data.csv"
  end

  get "/data/weekly.csv" do
    data = Commit.filter(:user => USER).group_and_count(:created_on).order(:created_on)

    @stats = Hash.new(0)
    data.each do |row|
      @stats[row.values[:created_on].strftime("%Y,%U")] += row.values[:count]
    end

    etag "data/weekly-#{Commit.max(:created_on)}"
    content_type "text/csv"
    erb :"weekly_data.csv"
  end

  get "/style.css" do
    content_type "text/css", :charset => "utf-8"
    less :style
  end

end