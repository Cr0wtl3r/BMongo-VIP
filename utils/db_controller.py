# import os
#
# from database_validator.database_validator import DatabaseValidator
# from database_validator.db_access import DBConnection
# from mongo_functions.base_clean import BaseClean
# from mongo_functions.base_create import BaseCreate
# from mongo_functions.change_tributation_for_ncm import ChangeTributationForNCM
# from mongo_functions.find_ids import FindIds
# from mongo_functions.inactive_products import InactiveProducts
# from mongo_functions.mei_able import MeiAble
# from mongo_functions.movimentations_clean import MovimentationsClean
# from mongo_functions.reg_digisat_clean import RegDigisatClean
#
# db_user = os.getenv("DB_USER")
# db_pass = os.getenv("DB_PASS")
# db_host = os.getenv("DB_HOST")
#
# db_connection = DBConnection(db_user, db_pass, db_host, 12220)
#
# inactive_products = InactiveProducts(db_connection, log)
# mei_able = MeiAble(db_connection, log)
# find_ids = FindIds(db_connection, log)
# movimentations_clean = MovimentationsClean(db_connection, log)
# change_tributation_for_ncm = ChangeTributationForNCM(db_connection, log)
# base_clean = BaseClean(db_connection, log)
# base_create = BaseCreate(db_connection, log)
# reg_digisat_clean = RegDigisatClean(db_connection, log)
# .database_validator = DatabaseValidator(db_connection, log)