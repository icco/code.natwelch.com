Code.controllers  do

  get :index do
    render :index
  end

  get "/data/commit.csv" do
    data = Commit.order(:created_on).where(:user => Commit::USER).count(:group=>:created_on)

    @stats = Hash.new(0)
    data.each do |row|
      @stats[row[0].strftime("%D")] += row[1]
    end

    etag "data/commit-#{Commit.maximum(:created_on)}"
    content_type "text/csv"
    erb :"commit_data.csv"
  end

  get "/data/weekly.csv" do
    year = params["year"] || Time.now.year.to_s

    data = Commit.order(:created_on).where(:user => Commit::USER).count(:group=>:created_on)

    @stats = Hash.new(0)
    ("01".."52").each {|week| @stats[week] = 0 }
    data.each do |row|
      if row[0].strftime("%Y") == '2012'
        @stats[row[0].strftime("%U")] += row[1]
      end
    end

    etag "data/weekly-#{Commit.maximum(:created_on)}"
    content_type "text/csv"
    erb :"weekly_data.csv"
  end

  get "/style.css" do
    content_type "text/css", :charset => "utf-8"
    less :style
  end

end
