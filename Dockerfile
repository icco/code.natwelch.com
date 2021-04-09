From ruby:3.0.0

WORKDIR /opt
COPY . .

ENV PORT 8080
ENV RACK_ENV production

RUN gem install bundler:1.17.3
RUN bundle install --system --without=test development

CMD bundle exec thin -R config.ru start -p $PORT
EXPOSE 8080
