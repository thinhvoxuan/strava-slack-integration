ARGS = $(filter-out $@,$(MAKECMDGOALS))

build:
	docker-compose build

dev_up:
	docker-compose up -d

logs:
	docker-compose logs -f

down:
	docker-compose down

dep: 
	docker-compose exec app dep ensure -update

dep-add: 
	docker-compose exec app dep ensure -add $(ARGS)

dep-status: 
	make dep status

dev-local: 
	realize start

deploy-production:
	docker build -f Dockerfile -t strava-slack/backend:$$(git rev-parse --short HEAD) --force-rm --no-cache .
	docker image prune -f --filter label=stage=builder
	sed "s/{{commit}}/$$(git rev-parse --short HEAD)/g" app-template.yml > docker-compose-app.yml
	docker-compose -f docker-compose-app.yml up -d --force-recreate  

%:
	@: