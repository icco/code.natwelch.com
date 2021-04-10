From ruby:3.0.0

WORKDIR /opt

ENV PORT 8080
ENV RACK_ENV production
EXPOSE 8080

# Pin bundler
RUN gem install bundler:2.2.16

COPY . .

RUN bundle config set --local system 'true'
RUN bundle config set --local without 'test development'
RUN bundle install

CMD bundle exec thin -R config.ru start -p $PORT
